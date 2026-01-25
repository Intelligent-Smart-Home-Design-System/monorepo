package devices

import (
	"log"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/domain"
	"github.com/fschuetz04/simgo"
)

// Освещение

// LampIn является структурой для передачи входных данных
type LampIn struct {
	TurnOn bool
}

// LampOut является структурой для возврата обработанных данных
type LampOut struct {
	time float64
}

// Lamp реализует интерфейс entities.Entity
type Lamp struct {
	id       string
	turnedOn bool
	delay    float64
	trigger  *simgo.Event
}

func NewLamp(id string, delay float64) *Lamp {
	return &Lamp{
		id:       id,
		turnedOn: false,
		delay:    delay,
	}
}

// GetInDataStruct возвращает *LampIn
func (l *Lamp) GetInDataStruct() domain.InData {
	return &LampIn{}
}

// GetOutDataStruct  возвращает *LampOut
func (l *Lamp) GetOutDataStruct() domain.OutData {
	return &LampOut{}
}

func (l *Lamp) GetProcessFunc() func(process simgo.Process, in domain.InData, out domain.OutData) {
	return l.Process
}

func (l *Lamp) Process(process simgo.Process, in domain.InData, out domain.OutData) {
	for {
		if l.trigger == nil || l.trigger.Triggered() {
			l.trigger = process.Simulation.Event()
		}

		process.Wait(l.trigger)

		inData, ok := in.(*LampIn)
		if !ok {
			log.Println("Cannot convert inData to LampIn")
		}

		outData, ok := out.(*LampOut)
		if !ok {
			log.Println("Cannot convert outData to LampOut")
		}

		l.HandleEvent(process, inData, outData)
	}
}

// HandleEvent реализует бизнес логику обработки сущности
func (l *Lamp) HandleEvent(process simgo.Process, inData *LampIn, outData *LampOut) {
	l.turnedOn = inData.TurnOn
	outData.time = process.Simulation.Now()
}

func (l *Lamp) GetID() string {
	return l.id
}
