package simulations

import (
	"context"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/converter"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/fetcher"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/sender"
)

// В пакете реализовано управление симуляциями через соответствующие компоненты.
// Пакет связывает логику компонентов (fetcher, sender, engine, ...) между собой и
// старается как можно меньше реализовывать логику самостоятельно.

// Simulations - структура, которая усправляет всеми симуляциями.
type Simulations struct {
	// TODO: client (websocket / http)
	fetcher          fetcher.Fetcher
	sender           sender.Sender
	IDToEngine       map[string]engine.Engine        // engineID <-> engine
	IDToEventInChan  map[string]chan api.EventInDTO  // engineID <-> канал для входящих событий
	IDToEventOutChan map[string]chan api.EventOutDTO // engineID <-> канал для исходящих событий
	IDToDependencies map[string][]api.ActionDTO      // engineID <-> слайс со структурами, описывающими зависимости между сущностями (кто кого тригерит)
}

// NewSimulation создает Simulations
func NewSimulation(fetcher fetcher.Fetcher, sender sender.Sender) *Simulations {
	return &Simulations{
		fetcher:          fetcher,
		sender:           sender,
		IDToEngine:       make(map[string]engine.Engine),
		IDToEventInChan:  make(map[string]chan api.EventInDTO),
		IDToEventOutChan: make(map[string]chan api.EventOutDTO),
		IDToDependencies: make(map[string][]api.ActionDTO),
	}
}

func (s *Simulations) Init(ctx context.Context) error {
	slog.Debug("Creating simulations data and starting components...")

	go func() {
		s.sender.Run()
	}()

	err := s.InitEngines()
	if err != nil {
		return err
	}

	s.GetEnginesInChan()
	s.GetEnginesOutChan()

	s.StartSending()
	s.StartEngines(ctx)

	return nil
}

// Run запускает сервис симуляции. Принимает контекст для graceful shutdown.
func (s *Simulations) Run(ctx context.Context) error {
	slog.Info("Simulations started!")

	for {
		select {
		case <-ctx.Done():
			s.Stop()
			return ctx.Err()
		default:
		}

		SimIDToEvents, err := s.fetcher.GetEvents() // должна быть блокирующая операция
		if err != nil {
			return err
		}

		for simID, events := range SimIDToEvents {
			for _, event := range events {
				s.IDToEventInChan[simID] <- event
			}
		}
	}
}

// InitEngines инициализирует движки для всех симуляций, заполняя их данными о поле и сущностях, полученными от fetcher.
func (s *Simulations) InitEngines() error {
	// TODO: понять когда приходят данные и как (в какой момент что инициализировать)
	simulationsID := s.fetcher.GetSimulationsID()
	for _, simID := range simulationsID {
		s.IDToEngine[simID] = engine.NewSimEngine()
	}

	IDToFields, err := s.fetcher.GetFields()
	if err != nil {
		return err
	}

	for simID, fieldData := range IDToFields {
		simField := converter.FieldFromDTO(fieldData)
		s.IDToEngine[simID].SetField(simField)
	}

	IDToEntitiesData, err := s.fetcher.GetEntities()
	if err != nil {
		return err
	}

	IDToDependencies, err := s.fetcher.GetDependencies()
	if err != nil {
		return err
	}

	for simID, entitiesData := range IDToEntitiesData {
		entities, err := converter.EntitiesFromDTO(entitiesData, s.IDToEngine[simID])
		if err != nil {
			return err
		}

		s.IDToEngine[simID].InitEntities(entities, IDToDependencies[simID])
		if s.IDToEngine[simID].CheckCircleDependencies() {
			slog.Error("Circle dependencies detected in simulation", "simulationID", simID)
		}

		s.IDToEngine[simID].InitProcesses()
	}

	return nil
}

// GetEnginesInChan возвращает каналы для входящих событий.
func (s *Simulations) GetEnginesInChan() {
	for id, engineItem := range s.IDToEngine {
		s.IDToEventInChan[id] = engineItem.GetInChan()
	}
}

// GetEnginesOutChan возвращает каналы для исходящих событий.
func (s *Simulations) GetEnginesOutChan() {
	for id, engineItem := range s.IDToEngine {
		s.IDToEventOutChan[id] = engineItem.GetOutChan()
	}
}

// StartSending запускает горутины для отправки событий из каналов исходящих событий в sender.
func (s *Simulations) StartSending() {
	for _, eventsOutCh := range s.IDToEventOutChan {
		go func(ch <-chan api.EventOutDTO) {
			for event := range ch {
				s.sender.AddEvent(event)
			}
		}(eventsOutCh)
	}
}

// StartEngines запускает горутины для каждого движка, чтобы они начали обрабатывать события.
func (s *Simulations) StartEngines(ctx context.Context) {
	for _, engineItem := range s.IDToEngine {
		go func(engineItem engine.Engine) {
			err := engineItem.Run(ctx)
			if err != nil {
				slog.Error("Error while starting engine", "error", err)
			}
		}(engineItem)
	}
}

// Stop закрывает каналы для входящих событий и останавливает симуляцию.
func (s *Simulations) Stop() {
	for _, eventInChan := range s.IDToEventInChan {
		close(eventInChan)
	}
}
