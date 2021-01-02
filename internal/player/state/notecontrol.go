package state

import (
	"time"

	"github.com/gotracker/gomixing/sampling"
	"github.com/gotracker/gomixing/volume"

	"gotracker/internal/player/intf"
	"gotracker/internal/player/note"
)

// NoteControl is an instance of the instrument on a particular output channel
type NoteControl struct {
	intf.NoteControl

	Instrument       intf.Instrument
	OutputChannelNum int
	Volume           volume.Volume
	Data             interface{}
	Filter           intf.Filter
	Playback         intf.Playback
	Period           note.Period
}

// GetSample returns the sample at position `pos` in the instrument
func (nc *NoteControl) GetSample(pos sampling.Pos) volume.Matrix {
	if inst := nc.Instrument; inst != nil {
		dry := inst.GetSample(nc, pos)
		if nc.Filter != nil {
			wet := nc.Filter.Filter(dry)
			return wet
		}
		return dry
	}
	return nil
}

// GetOutputChannelNum returns the note-control's output channel number
func (nc *NoteControl) GetOutputChannelNum() int {
	return nc.OutputChannelNum
}

// GetInstrument returns the instrument that's on this instance
func (nc *NoteControl) GetInstrument() intf.Instrument {
	return nc.Instrument
}

// Attack sets the key on flag for the instrument
func (nc *NoteControl) Attack() {
	if inst := nc.Instrument; inst != nil {
		inst.Attack(nc)
	}
}

// Release clears the key on flag for the instrument
func (nc *NoteControl) Release() {
	if inst := nc.Instrument; inst != nil {
		inst.Release(nc)
	}
}

// NoteCut cuts the current playback of the instrument
func (nc *NoteControl) NoteCut() {
	if inst := nc.Instrument; inst != nil {
		inst.NoteCut(nc)
	}
}

// GetKeyOn gets the key on flag for the instrument
func (nc *NoteControl) GetKeyOn() bool {
	if inst := nc.Instrument; inst != nil {
		return inst.GetKeyOn(nc)
	}
	return false
}

// Update advances time by the amount specified by `tickDuration`
func (nc *NoteControl) Update(tickDuration time.Duration) {
	if inst := nc.Instrument; inst != nil {
		inst.Update(nc, tickDuration)
	}
}

// SetFilter sets the active filter on the instrument (which should be the same as what's on the channel)
func (nc *NoteControl) SetFilter(filter intf.Filter) {
	nc.Filter = filter
}

// SetVolume sets the active note-control's volume
func (nc *NoteControl) SetVolume(vol volume.Volume) {
	nc.Volume = vol
}

// GetVolume sets the active note-control's volume
func (nc *NoteControl) GetVolume() volume.Volume {
	return nc.Volume
}

// SetPeriod sets the active note-control's period
func (nc *NoteControl) SetPeriod(period note.Period) {
	nc.Period = period
}

// GetPeriod gets the active note-control's period
func (nc *NoteControl) GetPeriod() note.Period {
	return nc.Period
}

// SetPlayback sets the playback interface for the note-control
func (nc *NoteControl) SetPlayback(pb intf.Playback) {
	nc.Playback = pb
}

// GetPlayback gets the playback interface for the note-control
func (nc *NoteControl) GetPlayback() intf.Playback {
	return nc.Playback
}

// SetData sets the data interface for the note-control
func (nc *NoteControl) SetData(data interface{}) {
	nc.Data = data
}

// GetData gets the data interface for the note-control
func (nc *NoteControl) GetData() interface{} {
	return nc.Data
}
