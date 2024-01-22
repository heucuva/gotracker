//go:build linux || pulseaudio
// +build linux pulseaudio

package device

import (
	"context"
	"errors"

	"github.com/gotracker/gomixing/mixing"
	"github.com/gotracker/gomixing/sampling"
	"github.com/gotracker/playback/output"

	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/gotracker/internal/output/device/pulseaudio"
	"github.com/gotracker/gotracker/internal/output/mixer"
)

const pulseaudioName = "pulseaudio"

type pulseaudioDevice struct {
	device
	mix      mixer.Mixer
	channels int
	sampFmt  sampling.Format
	pa       *pulseaudio.Client
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
	panmixer := mixing.GetPanMixer(d.channels)
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
			mixedData := d.mix.Flatten(panmixer, row.SamplesLen, row.Data, row.MixerVolume, d.sampFmt)
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
