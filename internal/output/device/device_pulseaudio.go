//go:build linux || pulseaudio
// +build linux pulseaudio

package device

import (
	"context"
	"errors"

	"github.com/gotracker/playback/mixing"
	"github.com/gotracker/playback/mixing/sampling"
	"github.com/gotracker/playback/output"

	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/gotracker/internal/output/device/pulseaudio"
)

const pulseaudioName = "pulseaudio"

type pulseaudioDevice struct {
	device
	mix     mixing.Mixer
	sampFmt sampling.Format
	pa      *pulseaudio.Client
}

func (pulseaudioDevice) GetKind() deviceCommon.Kind {
	return deviceCommon.KindSoundCard
}

// Name returns the device name
func (pulseaudioDevice) Name() string {
	return pulseaudioName
}

func newPulseAudioDevice(settings deviceCommon.Settings) (Device, error) {
	d := pulseaudioDevice{
		device: device{
			onRowOutput: settings.OnRowOutput,
		},
		mix: mixing.Mixer{
			Channels: settings.Channels,
		},
	}

	switch settings.BitsPerSample {
	case 8:
		d.sampFmt = sampling.Format8BitUnsigned
	case 16:
		d.sampFmt = sampling.Format16BitLESigned
	}

	play, err := pulseaudio.New("Music", settings.SamplesPerSecond, settings.Channels, settings.BitsPerSample)
	if err != nil {
		return nil, err
	}

	d.pa = play
	return &d, nil
}

// Play starts the wave output device playing
func (d *pulseaudioDevice) Play(in <-chan *output.PremixData) error {
	return d.PlayWithCtx(context.Background(), in)
}

// PlayWithCtx starts the wave output device playing
func (d *pulseaudioDevice) PlayWithCtx(ctx context.Context, in <-chan *output.PremixData) error {
	panmixer := mixing.GetPanMixer(d.mix.Channels)
	if panmixer == nil {
		return errors.New("invalid pan mixer - check channel count")
	}

	myCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-myCtx.Done():
			return myCtx.Err()
		case row, ok := <-in:
			if !ok {
				return nil
			}
			mixedData := d.mix.Flatten(row.SamplesLen, row.Data, row.MixerVolume, d.sampFmt)
			d.pa.Output(mixedData)
			if d.onRowOutput != nil {
				d.onRowOutput(deviceCommon.KindSoundCard, row)
			}
		}
	}
}

// Close closes the wave output device
func (d *pulseaudioDevice) Close() error {
	if d.pa != nil {
		return d.pa.Close()
	}
	return nil
}

func init() {
	Map[pulseaudioName] = deviceDetails{
		create: newPulseAudioDevice,
		Kind:   deviceCommon.KindSoundCard,
	}
}
