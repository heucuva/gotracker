package effect

import (
	"fmt"
	"gotracker/internal/player/intf"
)

// PortaUp defines a portamento up effect
type PortaUp uint8 // 'F'

// PreStart triggers when the effect enters onto the channel state
func (e PortaUp) PreStart(cs intf.Channel, ss intf.Song) {
}

// Start triggers on the first tick, but before the Tick() function is called
func (e PortaUp) Start(cs intf.Channel, ss intf.Song) {
	cs.ResetRetriggerCount()
	cs.UnfreezePlayback()
}

// Tick is called on every tick
func (e PortaUp) Tick(cs intf.Channel, ss intf.Song, currentTick int) {
	xx := cs.GetEffectSharedMemory(uint8(e))

	if currentTick != 0 {
		doPortaUp(cs, float32(xx), 4)
	}
}

// Stop is called on the last tick of the row, but after the Tick() function is called
func (e PortaUp) Stop(cs intf.Channel, ss intf.Song, lastTick int) {
}

func (e PortaUp) String() string {
	return fmt.Sprintf("F%0.2x", uint8(e))
}