package devices

import (
	"time"

	"github.com/fschuetz04/simgo"
)

// Переключатели / розетки

type LampSwitcher struct {
	id       string
	turnedOn bool
	delay    time.Duration
	trigger  *simgo.Event
}

func (l *LampSwitcher) GetReactionDelay() time.Duration {
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

		switch l.turnedOn {
		case false:
			l.turnedOn = true
		case true:
			l.turnedOn = false
		}
	}
}
