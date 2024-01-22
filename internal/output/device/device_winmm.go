//go:build windows
// +build windows

package device

import (
	"context"
	"errors"
	"time"

	"github.com/gotracker/gomixing/mixing"
	"github.com/gotracker/gomixing/sampling"
	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/gotracker/internal/output/mixer"
	"github.com/gotracker/playback/output"
	winmm "github.com/heucuva/go-winmm"
)

const winmmName = "winmm"

type winmmDevice struct {
	device
	mix      mixer.Mixer
	channels int
	sampFmt  sampling.Format
	waveout  *winmm.WaveOut
}

func (winmmDevice) GetKind() deviceCommon.Kind {
	return deviceCommon.KindSoundCard
}

// Name returns the device name
func (winmmDevice) Name() string {
	return winmmName
}

func newWinMMDevice(settings deviceCommon.Settings) (Device, error) {
	d := winmmDevice{
		device: device{
			onRowOutput: settings.OnRowOutput,
		},
		mix:      settings.Mixer,
		channels: settings.Channels,
	}

	if d.mix == nil {
		d.mix = mixing.Mixer{
			Channels: settings.Channels,
		}
	}

	switch settings.BitsPerSample {
	case 8:
		d.sampFmt = sampling.Format8BitUnsigned
	case 16:
		d.sampFmt = sampling.Format16BitLESigned
	}

	var err error
	d.waveout, err = winmm.New(settings.Channels, settings.SamplesPerSecond, settings.BitsPerSample)
	if err != nil {
		return nil, err
	}
	if d.waveout == nil {
		return nil, errors.New("could not create winmm device")
	}
	return &d, nil
}

// Play starts the wave output device playing
func (d *winmmDevice) Play(in <-chan *output.PremixData) error {
	return d.PlayWithCtx(context.Background(), in)
}

// PlayWithCtx starts the wave output device playing
func (d *winmmDevice) PlayWithCtx(ctx context.Context, in <-chan *output.PremixData) error {
	type RowWave struct {
		Wave *winmm.WaveOutData
		Row  *output.PremixData
	}

	panmixer := mixing.GetPanMixer(d.channels)
	if panmixer == nil {
		return errors.New("invalid pan mixer - check channel count")
	}

	myCtx, cancel := context.WithCancel(ctx)

	out := make(chan RowWave, 3)

	go func() {
		defer cancel()
		defer close(out)
		for {
			select {
			case <-myCtx.Done():
				return
			case row, ok := <-in:
				if !ok {
					return
				}
				mixedData := d.mix.Flatten(panmixer, row.SamplesLen, row.Data, row.MixerVolume, d.sampFmt)
				rowWave := RowWave{
					Wave: d.waveout.Write(mixedData),
					Row:  row,
				}
				out <- rowWave
			}
		}
	}()

	for {
		select {
		case <-myCtx.Done():
			return myCtx.Err()
		case rowWave, ok := <-out:
			if !ok {
				// done!
				return nil
			}
			if d.onRowOutput != nil {
				d.onRowOutput(deviceCommon.KindSoundCard, rowWave.Row)
			}
			for !d.waveout.IsHeaderFinished(rowWave.Wave) {
				time.Sleep(time.Microsecond * 1)
			}
		}
	}
}

// Close closes the wave output device
func (d *winmmDevice) Close() error {
	if d.waveout != nil {
		d.waveout.Close()
	}
	return nil
}

func init() {
	Map[winmmName] = deviceDetails{
		create: newWinMMDevice,
		Kind:   deviceCommon.KindSoundCard,
	}
}
