package simulations

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/converter"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
)

// Simulations - структура, которая усправляет всеми движками.
type Simulations struct {
	mu       sync.RWMutex
	sessions map[string]*simulationSession
}

type simulationSession struct {
	mu      sync.Mutex
	engine  engine.Engine
	stopped bool
}

// NewSimulation создает Simulations
func NewSimulation() *Simulations {
	return &Simulations{
		sessions: make(map[string]*simulationSession),
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
	if err := validateDependencies(entities, dependencies); err != nil {
		return err
	}
	eng.InitEntities(entities, dependencies)

	if eng.CheckCircleDependencies() {
		return errors.New("circle dependencies detected")
	}

	eng.InitProcesses()

	eng.InitStep()

	s.mu.Lock()
	if _, exists := s.sessions[reqID]; exists {
		s.mu.Unlock()
		eng.Stop()
		return fmt.Errorf("simulation %q already exists", reqID)
	}
	s.sessions[reqID] = &simulationSession{engine: eng}
	s.mu.Unlock()

	return nil
}

func validateDependencies(
	entitiesByID map[string]entities.Entity,
	dependencies map[string][]api.EdgeDTO,
) error {
	for sourceID, edges := range dependencies {
		if _, ok := entitiesByID[sourceID]; !ok {
			return fmt.Errorf("scenario source entity %q does not exist", sourceID)
		}
		for _, edge := range edges {
			if _, ok := entitiesByID[edge.ToID]; !ok {
				return fmt.Errorf("scenario target entity %q does not exist", edge.ToID)
			}
		}
	}
	return nil
}

// Tick продвигает симуляцию на один шаг.
// Вызывается при получении simulation:tick от клиента.
func (s *Simulations) Tick(reqID string, payload api.SimulationTickPayload) (*api.SimulationStepPayload, error) {
	s.mu.RLock()
	session, ok := s.sessions[reqID]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("simulation not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	if session.stopped {
		return nil, errors.New("simulation not found")
	}

	for index, input := range payload.Inputs {
		if err := session.engine.HandleEvent(normalizeInput(input)); err != nil {
			return nil, fmt.Errorf("input %d: %w", index, err)
		}
	}

	if err := session.engine.Step(); err != nil {
		return nil, err
	}

	return session.engine.CollectStep(payload.Tick), nil
}

func normalizeInput(input api.EventDTO) api.EventDTO {
	if len(input.Payload) == 0 {
		return input
	}

	var meta struct {
		Trigger string `json:"trigger"`
	}

	if err := json.Unmarshal(input.Payload, &meta); err != nil {
		return input
	}

	if meta.Trigger != "" {
		input.EntityID = meta.Trigger
	}

	return input
}

// Stop останавливает и удаляет движок симуляции.
// Вызывается при получении simulation:stop от клиента или разрыве соединения.
func (s *Simulations) Stop(reqID string) error {
	s.mu.Lock()
	session, ok := s.sessions[reqID]
	if !ok {
		s.mu.Unlock()
		return errors.New("simulation not found")
	}
	delete(s.sessions, reqID)
	s.mu.Unlock()

	session.mu.Lock()
	defer session.mu.Unlock()
	if session.stopped {
		return errors.New("simulation not found")
	}
	session.stopped = true
	session.engine.Stop()

	return nil
}
