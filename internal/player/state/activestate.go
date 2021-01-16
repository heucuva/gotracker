package state

import (
	"time"

	"github.com/gotracker/gomixing/mixing"
	"github.com/gotracker/gomixing/sampling"
	"github.com/gotracker/gomixing/volume"

	"gotracker/internal/player/intf"
	"gotracker/internal/player/note"
)

// activeState is the active state of a channel
type activeState struct {
	playbackState
	VoiceActive bool
	NoteControl intf.NoteControl
	PeriodDelta note.PeriodDelta
}

// Reset sets the active state to defaults
func (a *activeState) Reset() {
	a.playbackState.Reset()
	a.VoiceActive = true
	a.NoteControl = nil
	a.PeriodDelta = 0
}

// Render renders an active channel's sample data for a the provided number of samples
func (a *activeState) Render(globalVolume volume.Volume, mix *mixing.Mixer, panmixer mixing.PanMixer, samplerSpeed float32, samples int, duration time.Duration) (*mixing.Data, error) {
	if a.Period == nil {
		return nil, nil
	}

	nc := a.NoteControl
	if nc == nil {
		return nil, nil
	}
	nc.SetVolume(a.Volume * globalVolume)
	period := a.Period.Add(a.PeriodDelta)
	nc.SetPeriod(period)

	samplerAdd := float32(period.GetSamplerAdd(float64(samplerSpeed)))

	nc.Update(duration)

	panning := nc.GetCurrentPanning()
	volMatrix := panmixer.GetMixingMatrix(panning)

	// make a stand-alone data buffer for this channel for this tick
	var data mixing.MixBuffer
	if a.VoiceActive {
		data = mix.NewMixBuffer(samples)
		mixData := mixing.SampleMixIn{
			Sample:    sampling.NewSampler(nc, a.Pos, samplerAdd),
			StaticVol: volume.Volume(1.0),
			VolMatrix: volMatrix,
			MixPos:    0,
			MixLen:    samples,
		}
		data.MixInSample(mixData)
	}

	a.Pos.Add(samplerAdd * float32(samples))

	return &mixing.Data{
		Data:       data,
		Pan:        a.Pan,
		Volume:     volume.Volume(1.0),
		SamplesLen: samples,
	}, nil
}