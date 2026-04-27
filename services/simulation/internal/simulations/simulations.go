package simulations

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/converter"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
)

// В пакете реализовано управление симуляциями через соответствующие компоненты.
// Пакет связывает логику компонентов (fetcher, sender, engine, ...) между собой и
// старается как можно меньше реализовывать логику самостоятельно.

// Simulations - структура, которая усправляет всеми симуляциями.
type Simulations struct {
	mu         sync.RWMutex
	IDToEngine map[string]engine.Engine        // engineID <-> engine
}

// NewSimulation создает Simulations
func NewSimulation() *Simulations {
	return &Simulations{
		IDToEngine: make(map[string]engine.Engine),
	}
}

// Start инициализирует и запускает движок для симуляции.
// Вызывается при получении simulation:start от клиента.
func (s *Simulations) Start(reqID string, payload api.SimulationStartPayload) error {
	eng := engine.NewSimEngine(payload.DtSim)
 
	simField := converter.FieldFromDTO(payload.Apartment)
	eng.SetField(simField)
 
	entities, err := converter.EntitiesFromDTO(payload.Devices, eng)
	if err != nil {
		return err
	}
	dependencies := converter.DependenciesFromDTO(payload.Scenarios)
	eng.InitEntities(entities, dependencies)
 
	if eng.CheckCircleDependencies() {
		return errors.New("circle dependencies detected")
	}
 
	eng.InitProcesses()
 
	go func() {
		if err := eng.Run(); err != nil {
			slog.Error("engine error", "reqID", reqID, "error", err)
		}
	}()
 
	s.mu.Lock()
	s.IDToEngine[reqID] = eng
	s.mu.Unlock()
 
	return nil
}

// Tick продвигает симуляцию на один шаг.
// Вызывается при получении simulation:tick от клиента.
func (s *Simulations) Tick(reqID string, payload api.SimulationTickPayload) (*api.SimulationStepPayload, error) {
	s.mu.RLock()
	eng, ok := s.IDToEngine[reqID]
	s.mu.RUnlock()

	if !ok {
		return nil, errors.New("simulation not found")
	}
 
	for _, input := range payload.Inputs {
		eng.GetInChan() <- converter.InputToEventDTO(input)
	}
 
	eng.Step()
 
	return eng.CollectStep(payload.Tick), nil
}

// Stop останавливает и удаляет движок симуляции.
// Вызывается при получении simulation:stop от клиента или разрыве соединения.
func (s *Simulations) Stop(reqID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	eng, ok := s.IDToEngine[reqID]
	if !ok {
		return errors.New("simulation not found")
	}
 
	eng.Stop()
	delete(s.IDToEngine, reqID)
 
	return nil
}
