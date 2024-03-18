//go:build windows && directsound
// +build windows,directsound

package device

import (
	"context"
	"errors"
	"io"

	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/playback/mixing"
	"github.com/gotracker/playback/mixing/sampling"
	"github.com/gotracker/playback/output"
	directsound "github.com/heucuva/go-directsound"
	win32 "github.com/heucuva/go-win32"
	winmm "github.com/heucuva/go-winmm"
	"golang.org/x/sys/windows"
)

const dsoundName = "directsound"

type dsoundDevice struct {
	device

	ds           *directsound.DirectSound
	lpdsbPrimary *directsound.Buffer
	wfx          *winmm.WAVEFORMATEX

	mix     mixing.Mixer
	sampFmt sampling.Format
}

// Name returns the device name
func (dsoundDevice) Name() string {
	return dsoundName
}

func (dsoundDevice) GetKind() deviceCommon.Kind {
	return deviceCommon.KindSoundCard
}

func newDSoundDevice(settings deviceCommon.Settings) (Device, error) {
	d := dsoundDevice{
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

	preferredDeviceName := ""
	ds, err := directsound.NewDSound(preferredDeviceName)
	if err != nil {
		return nil, err
	}
	d.ds = ds
	if d.ds == nil {
		return nil, errors.New("could not create directsound device")
	}

	lpdsbPrimary, wfx, err := ds.CreateSoundBufferPrimary(settings.Channels, settings.SamplesPerSecond, settings.BitsPerSample)
	if err != nil {
		ds.Close()
		return nil, err
	}
	d.lpdsbPrimary = lpdsbPrimary
	d.wfx = wfx

	return &d, nil
}

// Play starts the wave output device playing
func (d *dsoundDevice) Play(in <-chan *output.PremixData) error {
	return d.PlayWithCtx(context.Background(), in)
}

type playbackData struct {
	event windows.Handle
	row   *output.PremixData
	pos   int
}

type playbackBuffer struct {
	buffer     *directsound.Buffer
	notify     *directsound.Notify
	rows       []playbackData
	maxSamples int
	writePos   int
}

func (p *playbackBuffer) Add(mix *mixing.Mixer, row *output.PremixData, pos int, size int, blockAlign int, panmixer mixing.PanMixer, format sampling.Format) (int, error) {
	remaining := p.maxSamples - p.writePos
	samples := row.SamplesLen - pos
	if samples >= remaining {
		samples = remaining
	}

	bufPos := p.writePos * blockAlign
	segments, err := p.buffer.Lock(bufPos, samples*blockAlign)
	if err != nil {
		return 0, err
	}
	writeSegs := [][]byte{}
	if pos > 0 {
		front := make([]byte, pos*blockAlign)
		writeSegs = append(writeSegs, front)
	}
	writeSegs = append(writeSegs, segments...)
	if samples < row.SamplesLen {
		rem := row.SamplesLen - samples
		rear := make([]byte, rem*blockAlign)
		writeSegs = append(writeSegs, rear)
	}
	mix.FlattenTo(writeSegs, panmixer.NumChannels(), row.SamplesLen, row.Data, row.MixerVolume, format)
	if err := p.buffer.Unlock(segments); err != nil {
		return 0, err
	}

	p.writePos += samples
	remaining -= samples
	if remaining <= 0 {
		err = io.EOF
	}
	return samples, err
}

// PlayWithCtx starts the wave output device playing
func (d *dsoundDevice) PlayWithCtx(ctx context.Context, in <-chan *output.PremixData) error {
	maxOutstanding := 3
	maxOutstandingEvents := 1000

	panmixer := mixing.GetPanMixer(d.mix.Channels)
	if panmixer == nil {
		return errors.New("invalid pan mixer - check channel count")
	}

	done := make(chan struct{}, 1)
	defer close(done)

	myCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := []windows.Handle{}
	availableEvents := make(chan windows.Handle, maxOutstandingEvents)
	defer func() {
		for _, event := range events {
			win32.CloseHandle(event)
		}
	}()

	playbackBuffers := make([]playbackBuffer, maxOutstanding)
	playbackBufferSize := int(float64(d.wfx.NSamplesPerSec) * 0.5)
	availableBuffers := make(chan *playbackBuffer, len(playbackBuffers))
	defer close(availableBuffers)
	for i := range playbackBuffers {
		lpdsb, err := d.ds.CreateSoundBufferSecondary(d.wfx, playbackBufferSize*int(d.wfx.NBlockAlign))
		if err != nil {
			return err
		}
		playbackBuffers[i] = playbackBuffer{
			buffer:     lpdsb,
			maxSamples: playbackBufferSize,
		}
		availableBuffers <- &playbackBuffers[i]
	}
	defer func() {
		for _, pb := range playbackBuffers {
			pb.buffer.Release()
		}
	}()

	getAvailableEvent := func() (windows.Handle, error) {
		select {
		case event := <-availableEvents:
			return event, nil
		default:
			event, err := win32.CreateEvent(nil, false, false, "")
			if err != nil {
				return event, err
			}
			events = append(events, event)
			return event, nil
		}
	}

	currentBuffer := <-availableBuffers

	out := make(chan *playbackBuffer, maxOutstanding)
	go func() {
		defer close(out)
		defer cancel()
		for {
			select {
			case <-myCtx.Done():
				return
			case row, ok := <-in:
				if !ok {
					return
				}

				size := row.SamplesLen
				pos := 0

				blockAlign := int(d.wfx.NBlockAlign)
				if size > 0 {
					event, err := getAvailableEvent()
					if err != nil {
						panic(err)
					}
					currentBuffer.rows = append(currentBuffer.rows, playbackData{
						event: event,
						row:   row,
						pos:   currentBuffer.writePos * blockAlign,
					})
				}
				for size > 0 {
					n, err := currentBuffer.Add(&d.mix, row, pos, row.SamplesLen, blockAlign, panmixer, d.sampFmt)
					size -= n
					pos += n
					if err != nil {
						if !errors.Is(err, io.EOF) {
							panic(err)
						}
						currentBuffer.writePos = 0
						out <- currentBuffer
						currentBuffer = <-availableBuffers
					}
				}
			}
		}
	}()
	for buffer := range out {
		endEvent, err := getAvailableEvent()
		if err != nil {
			return err
		}
		if err := d.playWaveBuffer(buffer, endEvent); err != nil {
			return err
		}
		for _, n := range buffer.rows {
			availableEvents <- n.event
		}
		availableEvents <- endEvent
		buffer.rows = []playbackData{}
		availableBuffers <- buffer
	}
	done <- struct{}{}
	return nil
}

func (d *dsoundDevice) playWaveBuffer(p *playbackBuffer, endEvent windows.Handle) error {
	notify, err := p.buffer.GetNotify()
	if err != nil {
		return err
	}
	defer notify.Release()

	pn := []directsound.PositionNotify{}

	for _, n := range p.rows {
		pn = append(pn, directsound.PositionNotify{
			Offset:      uint32(n.pos),
			EventNotify: n.event,
		})
	}

	pn = append(pn, directsound.PositionNotify{
		Offset:      directsound.DSBPN_OFFSETSTOP,
		EventNotify: endEvent,
	})

	if err := notify.SetNotificationPositions(pn); err != nil {
		return err
	}

	// play (non-looping)
	if err := p.buffer.Play(false); err != nil {
		return err
	}

	for _, n := range p.rows {
		if d.onRowOutput != nil {
			d.onRowOutput(deviceCommon.KindSoundCard, n.row)
		}
		if err := win32.WaitForSingleObjectInfinite(n.event); err != nil {
			return err
		}
	}
	return win32.WaitForSingleObjectInfinite(endEvent)
}

// Close closes the wave output device
func (d *dsoundDevice) Close() error {
	if d.lpdsbPrimary != nil {
		d.lpdsbPrimary.Release()
	}
	if d.ds != nil {
		d.ds.Close()
	}
	return nil
}

func init() {
	Map[dsoundName] = deviceDetails{
		create: newDSoundDevice,
		Kind:   deviceCommon.KindSoundCard,
	}
}
