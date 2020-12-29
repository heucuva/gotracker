package effect

import (
	"fmt"

	"gotracker/internal/player/intf"
)

// RowJump defines a row jump effect
type RowJump uint8 // 'D'

// PreStart triggers when the effect enters onto the channel state
func (e RowJump) PreStart(cs intf.Channel, p intf.Playback) {
}

// Start triggers on the first tick, but before the Tick() function is called
func (e RowJump) Start(cs intf.Channel, p intf.Playback) {
	cs.ResetRetriggerCount()
}

// Tick is called on every tick
func (e RowJump) Tick(cs intf.Channel, p intf.Playback, currentTick int) {
}

// Stop is called on the last tick of the row, but after the Tick() function is called
func (e RowJump) Stop(cs intf.Channel, p intf.Playback, lastTick int) {
	xy := uint8(e)
	x := xy >> 4
	y := xy & 0x0f
	row := intf.RowIdx(x*10 + y)
	p.SetNextRow(row)
}

func (e RowJump) String() string {
	return fmt.Sprintf("D%0.2x", uint8(e))
}
