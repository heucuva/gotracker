package output

import (
	"bufio"
	"gotracker/internal/player/render"
	"gotracker/internal/player/render/mixer"
	"os"

	"github.com/pkg/errors"
)

type fileDeviceWav struct {
	device
	channels      int
	bitsPerSample int
	mix           mixer.Mixer
}

type fileInternalWav struct {
	f  *os.File
	w  *bufio.Writer
	sz uint32
}

const (
	fileChunkSizePos     = 4
	fileSubchunk2SizePos = 40
)

func newFileWavDevice(settings Settings) (Device, error) {
	fd := fileDeviceWav{
		channels:      settings.Channels,
		bitsPerSample: settings.BitsPerSample,
	}
	f, err := os.OpenFile(settings.Filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0)
	if err != nil {
		return nil, err
	}

	if f == nil {
		return nil, errors.New("unexpected file error")
	}

	w := bufio.NewWriter(f)
	// RIFF header
	w.Write([]byte{'R', 'I', 'F', 'F'}) // ChunkID
	w.Write([]byte{0, 0, 0, 0})         // ChunkSize
	w.Write([]byte{'W', 'A', 'V', 'E'}) // Format

	// fmt header
	w.Write([]byte{'f', 'm', 't', ' '})                                      // Subchunk1ID
	w.Write([]byte{16, 0, 0, 0})                                             // Subchunk1Size
	w.Write([]byte{1, 0})                                                    // AudioFormat (1 = PCM)
	w.Write([]byte{uint8(settings.Channels), uint8(settings.Channels >> 8)}) // NumChannels
	w.Write([]byte{uint8(settings.SamplesPerSecond), uint8(settings.SamplesPerSecond >> 8),
		uint8(settings.SamplesPerSecond >> 16), uint8(settings.SamplesPerSecond >> 24)}) // SampleRate
	byteRate := settings.SamplesPerSecond * settings.Channels * settings.BitsPerSample / 8
	w.Write([]byte{uint8(byteRate), uint8(byteRate >> 8), uint8(byteRate >> 16), uint8(byteRate >> 24)}) // ByteRate
	blockAlign := settings.Channels * settings.BitsPerSample / 8
	w.Write([]byte{uint8(blockAlign), uint8(blockAlign >> 8)})                         // BlockAlign
	w.Write([]byte{uint8(settings.BitsPerSample), uint8(settings.BitsPerSample >> 8)}) // BitsPerSample

	// data header
	w.Write([]byte{'d', 'a', 't', 'a'}) // Subchunk2ID
	w.Write([]byte{0, 0, 0, 0})         // Subchunk2Size

	fd.internal = &fileInternalWav{
		f: f,
		w: w,
	}

	return &fd, nil
}

// Play starts the wave output device playing
func (d *fileDeviceWav) Play(in <-chan render.RowRender) {
	i := (d.internal.(*fileInternalWav))
	panmixer := mixer.GetPanMixer(d.channels)
	for row := range in {
		data := d.mix.NewMixBuffer(d.channels, row.SamplesLen)
		for _, rdata := range row.RenderData {
			pos := 0
			for _, cdata := range rdata {
				if cdata.Flush != nil {
					cdata.Flush()
				}
				if len(cdata.Data) > 0 {
					volMtx := cdata.Volume.Apply(panmixer.GetMixingMatrix(cdata.Pan)...)
					data.Add(pos, cdata.Data, volMtx)
				}
				pos += cdata.SamplesLen
			}
		}
		mixedData := data.ToRenderData(row.SamplesLen, d.bitsPerSample, len(row.RenderData))
		i.w.Write(mixedData)
		i.sz += uint32(len(mixedData))
	}
}

// Close closes the wave output device
func (d *fileDeviceWav) Close() {
	i := (d.internal.(*fileInternalWav))

	i.w.Flush()
	i.w = nil
	i.f.Seek(fileChunkSizePos, 0)
	chunkSize := 36 + i.sz
	i.f.Write([]byte{uint8(chunkSize), uint8(chunkSize >> 8), uint8(chunkSize >> 16), uint8(chunkSize >> 24)}) // ChunkSize
	i.f.Seek(fileSubchunk2SizePos, 0)
	i.f.Write([]byte{uint8(i.sz), uint8(i.sz >> 8), uint8(i.sz >> 16), uint8(i.sz >> 24)}) // Subchunk2Size
	i.f.Close()
}

func init() {
	fileDeviceMap[".wav"] = newFileWavDevice
}
