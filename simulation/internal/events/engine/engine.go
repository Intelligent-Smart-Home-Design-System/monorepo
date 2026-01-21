package engine

import (
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/simulation/internal/entities"
	"github.com/fschuetz04/simgo"
)

type SimEngine struct {
	simulation *simgo.Simulation
	started    bool
	// triggers хранит ключ = ID сущности, которая тригерит и значение = слайс сущностей, которые тригерятся
	triggers map[string][]entities.Entity
	// devices хранит ID девайса -> Девайс.
	devices map[string]entities.Entity
}

// NewSimEngine создает SimEngine
func NewSimEngine() *SimEngine {
	return &SimEngine{
		simulation: simgo.NewSimulation(),
		started:    false,
		triggers:   make(map[string][]entities.Entity),
		devices:    make(map[string]entities.Entity),
	}
}

func InitProcesses() {
	// проходимся for по массиву сущностей и запускаем процессы
}

func (s *SimEngine) SetStartTime(time time.Duration) {
	//if s.started == false {
	//	proc := s.simulation.Process(func(proc simgo.Process) {
	//		s.started = true
	//	})
	//	ev := proc.Simulation.Event()
	//	proc.Wait(ev)
	//}

	// учитываем начало симуляции (надо синхронизировать время simgo и настоящее)
}

func (s *SimEngine) Run() {
	// получаем новые ивенты и каждый новый ивент закидываем в HandleEvent
}

// HandleEvent обрабатывает event по его entityID
func (s *SimEngine) HandleEvent(entityID string) {
	entityTriggers := s.triggers[entityID]
	for _, entity := range entityTriggers {
		_, ok := s.devices[entityID]
		if ok {
			// если девайс, то должны учесть задержку + бизнес логика
		}
		// устанавливаем Event (если нет) и вызываем trigger
		entity.SetEvent()
		// учитываем задержку получателя, если устройство
		entity.Trigger()
	}
}
