//go:build flac
// +build flac

package file

import (
	"bufio"
	"context"
	"errors"
	"os"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"

	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/playback/mixing"
	"github.com/gotracker/playback/output"
)

type fileFlac struct {
	mix              mixing.Mixer
	samplesPerSecond int
	bitsPerSample    int

	f *os.File
	w *bufio.Writer
}

func newFileFlacDevice(settings deviceCommon.Settings) (File, error) {
	fd := fileFlac{
		mix: mixing.Mixer{
			Channels: settings.Channels,
		},
		samplesPerSecond: settings.SamplesPerSecond,
		bitsPerSample:    settings.BitsPerSample,
	}
	f, err := os.OpenFile(settings.Filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	if f == nil {
		return nil, errors.New("unexpected file error")
	}

	fd.f = f

	return &fd, nil
}

// Play starts the wave output device playing
func (d *fileFlac) Play(in <-chan *output.PremixData, onWrittenCallback WrittenCallback) error {
	return d.PlayWithCtx(context.Background(), in, onWrittenCallback)
}

// PlayWithCtx starts the wave output device playing
func (d *fileFlac) PlayWithCtx(ctx context.Context, in <-chan *output.PremixData, onWrittenCallback WrittenCallback) error {
	w := bufio.NewWriter(d.f)
	d.w = w
	// Encode FLAC stream.
	si := &meta.StreamInfo{
		BlockSizeMin:  16,
		BlockSizeMax:  65535,
		SampleRate:    uint32(d.samplesPerSecond),
		NChannels:     uint8(d.mix.Channels),
		BitsPerSample: uint8(d.bitsPerSample),
	}
	enc, err := flac.NewEncoder(w, si)
	if err != nil {
		return err
	}
	defer enc.Close()

	panmixer := mixing.GetPanMixer(d.mix.Channels)
	if panmixer == nil {
		return errors.New("invalid pan mixer - check channel count")
	}

	var channels frame.Channels
	switch d.mix.Channels {
	case 1:
		channels = frame.ChannelsMono
	case 2:
		channels = frame.ChannelsLR
	case 4:
		channels = frame.ChannelsLRLsRs
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
			mixedData := d.mix.FlattenToInts(panmixer.NumChannels(), row.SamplesLen, d.bitsPerSample, row.Data, row.MixerVolume)
			subframes := make([]*frame.Subframe, d.mix.Channels)
			for i := range subframes {
				subframe := &frame.Subframe{
					SubHeader: frame.SubHeader{
						Pred: frame.PredVerbatim,
					},
					Samples:  mixedData[i],
					NSamples: row.SamplesLen,
				}
				subframes[i] = subframe
			}
			for _, subframe := range subframes {
				sample := subframe.Samples[0]
				constant := true
				for _, s := range subframe.Samples[1:] {
					if sample != s {
						constant = false
					}
				}
				if constant {
					subframe.SubHeader.Pred = frame.PredConstant
				}
			}

			fr := &frame.Frame{
				Header: frame.Header{
					HasFixedBlockSize: false,
					BlockSize:         uint16(row.SamplesLen),
					SampleRate:        uint32(d.samplesPerSecond),
					Channels:          channels,
					BitsPerSample:     uint8(d.bitsPerSample),
				},
				Subframes: subframes,
			}
			if err := enc.WriteFrame(fr); err != nil {
				return err
			}
			if onWrittenCallback != nil {
				onWrittenCallback(row)
			}
		}
	}
}

// Close closes the wave output device
func (d *fileFlac) Close() error {
	if err := d.w.Flush(); err != nil {
		return err
	}
	d.w = nil
	return d.f.Close()
}

func init() {
	fileDeviceMap[".flac"] = newFileFlacDevice
}
