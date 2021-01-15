package intf

import (
	"time"

	"github.com/gotracker/gomixing/panning"
	"github.com/gotracker/gomixing/sampling"
	"github.com/gotracker/gomixing/volume"

	"gotracker/internal/player/note"
)

// InstrumentID is an identifier for an instrument/sample that means something to the format
type InstrumentID interface {
	IsEmpty() bool
}

// Instrument is an interface for instrument/sample data
type Instrument interface {
	IsInvalid() bool
	GetC2Spd() note.C2SPD
	SetC2Spd(note.C2SPD)
	GetDefaultVolume() volume.Volume
	GetID() InstrumentID
	GetSemitoneShift() int8
	InstantiateOnChannel(*OutputChannel) NoteControl
	SetFinetune(note.Finetune)
	GetFinetune() note.Finetune

	GetSample(NoteControl, sampling.Pos) volume.Matrix
	GetCurrentPanning(NoteControl) panning.Position
	Attack(NoteControl)
	Release(NoteControl)
	NoteCut(NoteControl)
	GetKeyOn(NoteControl) bool
	Update(NoteControl, time.Duration)
	SetEnvelopePosition(NoteControl, int)
}
