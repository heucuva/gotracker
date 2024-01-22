package mixer

import (
	"github.com/gotracker/gomixing/mixing"
	"github.com/gotracker/gomixing/sampling"
	"github.com/gotracker/gomixing/volume"
)

type Mixer interface {
	NewMixBuffer(samples int) mixing.MixBuffer
	Flatten(panmixer mixing.PanMixer, samplesLen int, row []mixing.ChannelData, mixerVolume volume.Volume, sampleFormat sampling.Format) []byte
	FlattenToInts(panmixer mixing.PanMixer, samplesLen, bitsPerSample int, row []mixing.ChannelData, mixerVolume volume.Volume) [][]int32
	FlattenTo(resultBuffers [][]byte, panmixer mixing.PanMixer, samplesLen int, row []mixing.ChannelData, mixerVolume volume.Volume, sampleFormat sampling.Format)
}

type sincMixer struct {
	mixing.Mixer
	sw []volume.Matrix
}

var coeffs = []float64{
	-0.005457697498514417,
	0.015877236320297066,
	-0.042223067860869427,
	0.069921667612960703,
	0.923763722852252167,
	0.069921667612960717,
	-0.042223067860869434,
	0.015877236320297073,
	-0.005457697498514417,
}

func NewMixer(channels int) Mixer {
	sw := make([]volume.Matrix, len(coeffs))
	for i := range sw {
		sw[i].Channels = channels
	}

	return &sincMixer{
		Mixer: mixing.Mixer{
			Channels: channels,
		},
		sw: sw,
	}
}

func (m sincMixer) Flatten(panmixer mixing.PanMixer, samplesLen int, row []mixing.ChannelData, mixerVolume volume.Volume, sampleFormat sampling.Format) []byte {
	data := m.NewMixBuffer(samplesLen)
	formatter := sampling.GetFormatter(sampleFormat)
	for _, rdata := range row {
		for _, cdata := range rdata {
			if cdata.Flush != nil {
				cdata.Flush()
			}
			if len(cdata.Data) > 0 {
				volMtx := panmixer.GetMixingMatrix(cdata.Pan).Apply(cdata.Volume)
				data.Add(cdata.Pos, &cdata.Data, volMtx)
			}
		}
	}

	m.applyFilter(data)

	return data.ToRenderData(samplesLen, m.Channels, mixerVolume, formatter)
}

// FlattenToInts runs a flatten on the channel data into separate channel data of int32 variety
// these int32s still respect the bitsPerSample size
func (m sincMixer) FlattenToInts(panmixer mixing.PanMixer, samplesLen, bitsPerSample int, row []mixing.ChannelData, mixerVolume volume.Volume) [][]int32 {
	data := m.NewMixBuffer(samplesLen)
	for _, rdata := range row {
		for _, cdata := range rdata {
			if cdata.Flush != nil {
				cdata.Flush()
			}
			if len(cdata.Data) > 0 {
				volMtx := panmixer.GetMixingMatrix(cdata.Pan).Apply(cdata.Volume)
				data.Add(cdata.Pos, &cdata.Data, volMtx)
			}
		}
	}

	m.applyFilter(data)

	return data.ToIntStream(panmixer.NumChannels(), samplesLen, bitsPerSample, mixerVolume)
}

// FlattenTo will to a final saturation mix of all the row's channel data into a single output buffer
func (m sincMixer) FlattenTo(resultBuffers [][]byte, panmixer mixing.PanMixer, samplesLen int, row []mixing.ChannelData, mixerVolume volume.Volume, sampleFormat sampling.Format) {
	data := m.NewMixBuffer(samplesLen)
	formatter := sampling.GetFormatter(sampleFormat)
	for _, rdata := range row {
		for _, cdata := range rdata {
			if cdata.Flush != nil {
				cdata.Flush()
			}
			if len(cdata.Data) > 0 {
				volMtx := panmixer.GetMixingMatrix(cdata.Pan).Apply(cdata.Volume)
				data.Add(cdata.Pos, &cdata.Data, volMtx)
			}
		}
	}

	m.applyFilter(data)

	data.ToRenderDataWithBufs(resultBuffers, samplesLen, mixerVolume, formatter)
}

func (m sincMixer) applyFilter(data []volume.Matrix) {
	for n := range data {
		sl := len(m.sw) - 1
		for j := 0; j < sl; j++ {
			m.sw[j] = m.sw[j+1]
		}
		m.sw[sl] = data[n]

		var sv volume.Matrix
		sv.Channels = m.Channels
		for j, h := range coeffs {
			v := m.sw[j].Apply(volume.Volume(h))
			sv.Accumulate(v)
		}

		data[n] = sv
	}
}
