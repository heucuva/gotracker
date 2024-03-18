package main_test

import (
	"errors"
	"testing"
	"time"

	"github.com/gotracker/playback/format"
	"github.com/gotracker/playback/output"
	"github.com/gotracker/playback/player/feature"
	"github.com/gotracker/playback/player/machine"
	"github.com/gotracker/playback/player/machine/settings"
	"github.com/gotracker/playback/player/sampler"
	"github.com/gotracker/playback/song"
	"github.com/heucuva/optional"
)

func BenchmarkPlayerS3M(b *testing.B) {
	var sd song.Data
	var sfmt format.Format
	var err error
	b.Run("load_s3m", func(b *testing.B) {
		sd, sfmt, err = format.Load("test/celestial_fantasia.s3m")
		if err != nil {
			b.Error(err)
		}
	})

	var us settings.UserSettings
	us.Reset()

	if err := sfmt.ConvertFeaturesToSettings(&us, []feature.Feature{feature.SongLoop{Count: 0}}); err != nil {
		b.Error(err)
	}

	pb, err := machine.NewMachine(sd, us)
	if err != nil {
		b.Error(err)
	}

	out := sampler.NewSampler(44100, 2, 0.5, nil)

	for err == nil {
		b.Run("generate_s3m", func(b *testing.B) {
			b.Helper()
			b.ReportAllocs()
			if err := pb.Tick(out); err != nil {
				if !errors.Is(err, song.ErrStopSong) {
					b.Error(err)
				}
				return
			}
		})
	}
}

func BenchmarkIT(b *testing.B) {
	var sd song.Data
	var sfmt format.Format
	var err error
	b.Run("load_it", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sd, sfmt, err = format.Load(".vscode/test/beyond_the_network.it")
			if err != nil {
				b.Error(err)
			}
		}
	})

	const (
		sampleRate       = 44100
		outputChannels   = 2
		stereoSeparation = 0.5

		stepDuration = time.Duration(sampleRate * outputChannels * 2)
	)

	var us settings.UserSettings
	us.Reset()

	if err := sfmt.ConvertFeaturesToSettings(&us, []feature.Feature{
		feature.SongLoop{Count: 0},
		feature.StartOrderAndRow{Order: optional.NewValue[int](38)},
	}); err != nil {
		b.Error(err)
	}

	pb, err := machine.NewMachine(sd, us)
	if err != nil {
		b.Error(err)
	}

	var step time.Duration
	out := sampler.NewSampler(sampleRate, outputChannels, stereoSeparation, func(premix *output.PremixData) {
		if premix != nil {
			step += time.Duration(premix.SamplesLen) / stepDuration
			b.SetBytes(int64(premix.SamplesLen))
		}
	})

	for err == nil {
		b.Run("generate_it", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := pb.Tick(out); err != nil {
					if !errors.Is(err, song.ErrStopSong) {
						b.Error(err)
					}
					return
				}
			}
		})
	}
}
