package engine

import (
	"context"
	"errors"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/fschuetz04/simgo"
)

const maxEventsBuffer = 1000

// SimEngine реализует интерфефс Engine
type SimEngine struct {
	simulation *simgo.Simulation // дискретная симуляция из simgo

	// IDToEntity хранит ключ = ID сущности, значение = структура сущности.
	IDToEntity map[string]entities.Entity

	// Поле для симуляции
	Field *field.Field

	// Канал для взодящих событий
	eventsInQueue chan api.EventInDTO

	// Канал для новых событий
	eventsOutQueue chan api.EventOutDTO
}

// NewSimEngine создает SimEngine
func NewSimEngine() *SimEngine {
	return &SimEngine{
		simulation:    simgo.NewSimulation(),
		IDToEntity:    make(map[string]entities.Entity),
		eventsInQueue: make(chan api.EventInDTO, maxEventsBuffer),
	}
}

func (s *SimEngine) GetOutChan() chan api.EventOutDTO {
	return s.eventsOutQueue
}

func (s *SimEngine) SetField(simField *field.Field) {
	s.Field = simField
}

// InitEntities создает сущности для симуляции из мапы с конфигом.
// IDToEntityType хранит ключ = ID сущности, значение = конфиг сущности.
func (s *SimEngine) InitEntities(IDToEntity map[string]entities.Entity) {
	s.IDToEntity = IDToEntity
}

// InitProcesses инициализирует данные для процессов и запускает процессы.
// Информация берется из map[string]entities.Entity, где ключ = ID сущности, значение = сущность.
// map[string]entities.Entity создается из конфига устройств (приходит из другого сервиса).
func (s *SimEngine) InitProcesses() {
	for _, entity := range s.IDToEntity {
		if entityWithProcess, ok := entity.(entities.EntityWithProcess); ok {
			s.simulation.ProcessReflect(entityWithProcess.GetProcessFunc())
		}
	}
}

func (s *SimEngine) GetInChan() chan api.EventInDTO {
	return s.eventsInQueue
}

func (s *SimEngine) Run(ctx context.Context) error {
	if s.simulation == nil {
		return errors.New("need simgo simulation for starting engine")
	} else if s.Field == nil {
		return errors.New("need field for starting engine")
	} else if s.eventsInQueue == nil {
		return errors.New("need queue for starting engine")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-s.eventsInQueue:
			if !ok {
				return nil
			}

			s.HandleEvent(event)
		}
	}
}

// HandleEvent обрабатывает event по его entityID
func (s *SimEngine) HandleEvent(event api.EventInDTO) {
	entity := s.IDToEntity[event.EntityID]
	receiversID := entity.GetReceiversID()

	for _, receiverID := range receiversID {
		s.eventsInQueue <- api.EventInDTO{ // может тормозить, можно сделать pending или semaphore
			EntityID: receiverID,
		}
	}

	if entityWithProcess, ok := s.IDToEntity[event.EntityID].(entities.EntityWithProcess); ok {
		err := entityWithProcess.HandleInDTO(event.Info)
		if err != nil {
			return
		}
	}
}

func (s *SimEngine) UpdateField(x, y int, cell field.Cell) error {
	if x < 0 || x > s.Field.Height {
		return errors.New("invalid parameter x")
	} else if y < 0 || y > s.Field.Width {
		return errors.New("invalid parameter y")
	}

	s.Field.Cells[x][y].Condition = cell.Condition

	return nil
}
