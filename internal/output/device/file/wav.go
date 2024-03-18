package file

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"os"

	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/playback/mixing"
	"github.com/gotracker/playback/mixing/sampling"
	"github.com/gotracker/playback/output"
)

type fileWav struct {
	mix     mixing.Mixer
	sampFmt sampling.Format

	f  *os.File
	w  *bufio.Writer
	sz uint32
}

const (
	wavFileChunkSizePos     = 4
	wavFileSubchunk2SizePos = 40
)

func newFileWavDevice(settings deviceCommon.Settings) (File, error) {
	fd := fileWav{
		mix: mixing.Mixer{
			Channels: settings.Channels,
		},
	}
	switch settings.BitsPerSample {
	case 8:
		fd.sampFmt = sampling.Format8BitSigned
	case 16:
		fd.sampFmt = sampling.Format16BitLESigned
	}

	f, err := os.OpenFile(settings.Filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	if f == nil {
		return nil, errors.New("unexpected file error")
	}

	byteRate := settings.SamplesPerSecond * settings.Channels * settings.BitsPerSample / 8
	blockAlign := settings.Channels * settings.BitsPerSample / 8

	w := bufio.NewWriter(f)
	// RIFF header
	if _, err := w.Write([]byte{'R', 'I', 'F', 'F'}); err != nil { // ChunkID
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(0)); err != nil { // ChunkSize
		return nil, err
	}
	if _, err := w.Write([]byte{'W', 'A', 'V', 'E'}); err != nil { // Format
		return nil, err
	}

	// fmt header
	if _, err := w.Write([]byte{'f', 'm', 't', ' '}); err != nil { // Subchunk1ID
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil { // Subchunk1Size
		return nil, err
	}
	// = win32.WAVEFORMATEX (before the CbSize)
	if err := binary.Write(w, binary.LittleEndian, uint16(0x001)); err != nil { // AudioFormat // = win32.WAVE_FORMAT_PCM
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(settings.Channels)); err != nil { // NumChannels
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(settings.SamplesPerSecond)); err != nil { // SampleRate
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(byteRate)); err != nil { // ByteRate
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(blockAlign)); err != nil { // BlockAlign
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(settings.BitsPerSample)); err != nil { // BitsPerSample
		return nil, err
	}

	// data header
	if _, err := w.Write([]byte{'d', 'a', 't', 'a'}); err != nil { // Subchunk2ID
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(0)); err != nil { // Subchunk2Size
		return nil, err
	}

	fd.f = f
	fd.w = w

	return &fd, nil
}

// Play starts the wave output device playing
func (d *fileWav) Play(in <-chan *output.PremixData, onWrittenCallback WrittenCallback) error {
	return d.PlayWithCtx(context.Background(), in, onWrittenCallback)
}

// PlayWithCtx starts the wave output device playing
func (d *fileWav) PlayWithCtx(ctx context.Context, in <-chan *output.PremixData, onWrittenCallback WrittenCallback) error {
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
			sz, err := d.w.Write(mixedData)
			if err != nil {
				return err
			}
			d.sz += uint32(sz)
			if onWrittenCallback != nil {
				onWrittenCallback(row)
			}
		}
	}
}

// Close closes the wave output device
func (d *fileWav) Close() error {
	d.w.Flush()
	chunkSize := 36 + d.sz
	if _, err := d.f.Seek(wavFileChunkSizePos, 0); err != nil {
		return err
	}
	if err := binary.Write(d.w, binary.LittleEndian, uint32(chunkSize)); err != nil { // ChunkSize
		return err
	}
	if _, err := d.f.Seek(wavFileSubchunk2SizePos, 0); err != nil {
		return err
	}
	if err := binary.Write(d.w, binary.LittleEndian, uint32(d.sz)); err != nil { // Subchunk2Size
		return err
	}
	if err := d.w.Flush(); err != nil {
		return err
	}
	d.w = nil
	return d.f.Close()
}

func init() {
	fileDeviceMap[".wav"] = newFileWavDevice
}
