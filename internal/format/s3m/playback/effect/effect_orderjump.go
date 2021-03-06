package effect

import (
	"fmt"

	"gotracker/internal/player/intf"
)

// OrderJump defines an order jump effect
type OrderJump uint8 // 'B'

// PreStart triggers when the effect enters onto the channel state
func (e OrderJump) PreStart(cs intf.Channel, ss intf.Song) {
	ss.SetCurrentOrder(uint8(e))
}

// Start triggers on the first tick, but before the Tick() function is called
func (e OrderJump) Start(cs intf.Channel, ss intf.Song) {
	cs.ResetRetriggerCount()
}

// Tick is called on every tick
func (e OrderJump) Tick(cs intf.Channel, ss intf.Song, currentTick int) {
}

// Stop is called on the last tick of the row, but after the Tick() function is called
func (e OrderJump) Stop(cs intf.Channel, ss intf.Song, lastTick int) {
}

func (e OrderJump) String() string {
	return fmt.Sprintf("B%0.2x", uint8(e))
}
