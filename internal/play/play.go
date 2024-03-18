package play

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	progressBar "github.com/cheggaaa/pb"
	"github.com/gotracker/playback"

	"github.com/gotracker/gotracker/internal/feature"
	"github.com/gotracker/gotracker/internal/logging"
	"github.com/gotracker/gotracker/internal/output"
	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/gotracker/internal/playlist"
	"github.com/gotracker/playback/format"
	itFeature "github.com/gotracker/playback/format/it/feature"
	playbackOutput "github.com/gotracker/playback/output"
	playbackFeature "github.com/gotracker/playback/player/feature"
	"github.com/gotracker/playback/player/machine"
	"github.com/gotracker/playback/player/machine/settings"
	"github.com/gotracker/playback/player/render"
	"github.com/gotracker/playback/player/sampler"
	"github.com/gotracker/playback/song"
	"github.com/gotracker/playback/tracing"
)

func Playlist(pl *playlist.Playlist, features []playbackFeature.Feature, settings *Settings, outCfg *deviceCommon.Settings, debugCfg *DebugSettings, logger logging.Log) (bool, error) {
	var (
		play      playback.Playback
		progress  *progressBar.ProgressBar
		lastOrder int
	)

	outCfg.OnRowOutput = func(kind deviceCommon.Kind, premix *playbackOutput.PremixData) {
		row := premix.Userdata.(*render.RowRender)
		switch kind {
		case deviceCommon.KindSoundCard:
			if row.RowText != nil {
				logger.Printf("[%0.3d:%0.3d] %s\n", row.Order, row.Row, row.RowText.String())
			}
		case deviceCommon.KindFile:
			if progress == nil {
				progress = progressBar.StartNew(play.GetNumOrders())
				lastOrder = row.Order
			}
			if lastOrder != row.Order {
				progress.Increment()
				lastOrder = row.Order
			}
		}
	}

	waveOut, features, err := output.CreateOutputDevice(*outCfg)
	if err != nil {
		return false, err
	}
	defer waveOut.Close()

	var (
		r  renderer
		wg sync.WaitGroup
	)
	defer r.Close()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := waveOut.Play(r.PremixData()); err != nil {
			switch {
			case errors.Is(err, song.ErrStopSong):
			case errors.Is(err, context.Canceled):

			default:
				log.Fatalln(err)
			}
		}
	}()

	features = append(features, playbackFeature.IgnoreUnknownEffect{Enabled: !debugCfg.PanicOnUnhandledEffect})

	if debugCfg.Tracing {
		features = append(features, feature.EnableTracing{
			Filename: debugCfg.TracingFile,
		})
	}

	logger.Printf("Output device: %s\n", waveOut.Name())

	err = r.renderSongs(pl, features, settings, outCfg, func(m machine.MachineTicker, outCfg *deviceCommon.Settings, out *sampler.Sampler, tickInterval time.Duration, tracer tracing.Tracer) error {
		defer func() {
			if progress != nil {
				progress.Set64(progress.Total)
				progress.Finish()
			}
		}()

		logger.Printf("Order Looping Enabled: %v\n", m.CanOrderLoop())
		logger.Printf("Song: %s\n", m.GetName())

		p, err := NewPlayer(context.TODO(), tickInterval)
		if err != nil {
			return err
		}

		if err := p.Play(m, out, tracer); err != nil {
			return err
		}

		if err := p.WaitUntilDone(); err != nil {
			logger.Println()
			logger.Println(err)
			return err
		}

		return nil
	})
	if !r.playedAtLeastOneEntry || err != nil {
		return r.playedAtLeastOneEntry, err
	}
	// force the close
	r.Close()

	wg.Wait()

	logger.Println()
	logger.Println("done!")

	return true, nil
}

func getFeatureByType[T playbackFeature.Feature](features []playbackFeature.Feature) (T, bool) {
	var empty T
	if len(features) == 0 {
		return empty, false
	}

	for _, f := range features {
		switch v := f.(type) {
		case T:
			return v, true
		}
	}

	return empty, false
}

type renderer struct {
	playedAtLeastOneEntry bool
	outBufs               chan *playbackOutput.PremixData
}

func (p *renderer) PremixData() <-chan *playbackOutput.PremixData {
	if p.outBufs == nil {
		p.outBufs = make(chan *playbackOutput.PremixData, 128)
	}
	return p.outBufs
}

func (p *renderer) Close() error {
	if p.outBufs != nil {
		close(p.outBufs)
		p.outBufs = nil
	}
	return nil
}

type playerCBFunc func(pb machine.MachineTicker, outCfg *deviceCommon.Settings, out *sampler.Sampler, tickInterval time.Duration, tracer tracing.Tracer) error

func (p *renderer) renderSongs(pl *playlist.Playlist, features []playbackFeature.Feature, renderSettings *Settings, outCfg *deviceCommon.Settings, startPlayingCB playerCBFunc) error {
	tickInterval := time.Duration(5) * time.Millisecond
	if setting, ok := getFeatureByType[feature.PlayerSleepInterval](features); ok {
		if setting.Enabled {
			tickInterval = setting.Interval
		} else {
			tickInterval = 0
		}
	}

	canPossiblyLoop := true
	if setting, ok := getFeatureByType[playbackFeature.SongLoop](features); ok {
		canPossiblyLoop = (setting.Count != 0)
	}

	out := sampler.NewSampler(outCfg.SamplesPerSecond, outCfg.Channels, float32(outCfg.StereoSeparation)/100.0, func(premix *playbackOutput.PremixData) {
		p.outBufs <- premix
	})
	if out == nil {
		return errors.New("could not setup playback sampler")
	}

	var us settings.UserSettings

	for _, feat := range features {
		switch f := feat.(type) {
		case feature.EnableTracing:
			if err := us.SetupTracingWithFilename(f.Filename); err != nil {
				return err
			}
		}
	}

	defer us.CloseTracing()

playlistLoop:
	for _, songIdx := range pl.GetPlaylist() {
		entry := pl.GetSong(songIdx)
		if entry == nil {
			continue
		}
		songData, songFmt, err := format.Load(entry.Filepath, features...)
		if err != nil {
			return fmt.Errorf("could not create song state: %w", err)
		}

		cfg := features

		cfg = append(cfg, playbackFeature.StartOrderAndRow{
			Order: entry.Start.Order,
			Row:   entry.Start.Row,
		})

		endOrder, endOrderSet := entry.End.Order.Get()
		endRow, endRowSet := entry.End.Row.Get()
		if endOrderSet && endRowSet && endOrder >= 0 && endRow >= 0 {
			cfg = append(cfg, playbackFeature.PlayUntilOrderAndRow{
				Order: endOrder,
				Row:   endRow,
			})
		}

		if tempo, ok := entry.Tempo.Get(); ok {
			cfg = append(cfg, playbackFeature.SetDefaultTempo{Tempo: tempo})
		}

		if bpm, ok := entry.BPM.Get(); ok {
			cfg = append(cfg, playbackFeature.SetDefaultBPM{BPM: bpm})
		}

		var loopCount int
		if canPossiblyLoop {
			if l, ok := entry.Loop.Count.Get(); ok {
				loopCount = l
			}
		}
		cfg = append(cfg,
			playbackFeature.SongLoop{Count: loopCount},
			itFeature.LongChannelOutput{Enabled: renderSettings.ITLongChannelOutput},
			itFeature.NewNoteActions{Enabled: renderSettings.ITEnableNNA})

		us.Reset()
		if songFmt != nil {
			if err := songFmt.ConvertFeaturesToSettings(&us, cfg); err != nil {
				return fmt.Errorf("could not configure playback settings: %w", err)
			}
		}

		playback, err := machine.NewMachine(songData, us)
		if err != nil {
			return fmt.Errorf("could not create playback machine: %w", err)
		}

		if err = startPlayingCB(playback, outCfg, out, tickInterval, us.Tracer); err != nil {
			continue
		}

		pl.MarkPlayed(entry)

		p.playedAtLeastOneEntry = true
	}

	if pl.IsLooping() {
		goto playlistLoop
	}

	return nil
}
