package devices

import (
	"github.com/fschuetz04/simgo"
)

// Переключатели / розетки

type LampSwitcher struct {
	id       string
	turnedOn bool
	delay    float64
	trigger  *simgo.Event
}

func (l *LampSwitcher) GetReactionDelay() float64 {
	return l.delay
}

func (l *LampSwitcher) Process(
	process simgo.Process,
) {
	for {
		if l.trigger.Triggered() {
			l.trigger = process.Simulation.Event()
		}
		process.Wait(l.trigger)

		process.Wait(process.Timeout(l.delay))

		switch l.turnedOn {
		case false:
			l.turnedOn = true
		case true:
			l.turnedOn = false
		}
	}
}
