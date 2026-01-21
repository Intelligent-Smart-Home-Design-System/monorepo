package devices

import (
	"log"
	"time"

	"github.com/fschuetz04/simgo"
)

// Освещение

type Lamp struct {
	id       string
	turnedOn bool
	delay    time.Duration
	trigger  *simgo.Event
}

func (l *Lamp) Process(process simgo.Process) {
	for {
		if l.trigger.Triggered() {
			l.trigger = process.Simulation.Event()
		}

		process.Wait(l.trigger)

		switch l.turnedOn {
		case true:
			l.turnedOn = false
			log.Printf("Lamp turned on at: %v", process.Simulation.Now())
		case false:
			l.turnedOn = true
			log.Printf("Lamp turned off at: %v", process.Simulation.Now())
		}
	}
}
