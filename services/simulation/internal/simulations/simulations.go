package simulations

import (
	"encoding/json"
	"errors"
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
	IDToEngine map[string]engine.Engine // engineID <-> engine
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

	simField, err := converter.ParseFloor(payload.Apartment)
	if err != nil {
		return err
	}

	eng.SetFloor(simField)

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

	eng.InitStep()

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
		input = normalizeInput(input)
		if input.Trigger != "" {
			input.EntityID = input.Trigger
		}
		eng.GetInChan() <- input
	}

	eng.Step()

	return eng.CollectStep(payload.Tick), nil
}

func normalizeInput(input api.EventInDTO) api.EventInDTO {
	if len(input.Payload) == 0 {
		return input
	}

	var meta struct {
		Kind    string `json:"kind"`
		Trigger string `json:"trigger"`
	}

	if err := json.Unmarshal(input.Payload, &meta); err != nil {
		return input
	}

	if input.Kind == "" {
		input.Kind = meta.Kind
	}
	if input.Trigger == "" {
		input.Trigger = meta.Trigger
	}

	return input
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
