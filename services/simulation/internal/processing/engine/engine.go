package engine

import (
	"errors"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
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

	// IDToEntityWithProcess хранит ключ = ID сущности с процессом, значение = структура сущности с процессом.
	IDToEntityWithProcess map[string]entities.EntityWithProcess

	// Поле для симуляции
	Field *field.Field

	// Канал для новых событий
	eventsQueue chan config.EventDTO
}

// NewSimEngine создает SimEngine
func NewSimEngine(fieldDTO config.FieldDTO) *SimEngine {
	s := &SimEngine{
		simulation:            simgo.NewSimulation(),
		IDToEntity:            make(map[string]entities.Entity),
		IDToEntityWithProcess: make(map[string]entities.EntityWithProcess),
		eventsQueue:           make(chan config.EventDTO, maxEventsBuffer),
	}

	simField := &field.Field{
		Width:  fieldDTO.Width,
		Height: fieldDTO.Height,
	}

	for i, cells := range fieldDTO.Cells {
		for j, cell := range cells {
			simField.Cells[i][j] = &field.Cell{
				X:            cell.X,
				Y:            cell.Y,
				Condition:    0,
				IsHiddenWall: false,
			}
		}
	}

	s.Field = simField

	return s
}

// InitEntities создает сущности для симуляции из мапы с конфигом.
// IDToEntityType хранит ключ = ID сущности, значение = конфиг сущности.
func (s *SimEngine) InitEntities(IDToBaseEntity map[string]entities.Entity, IDToEntityWithProcess map[string]entities.EntityWithProcess) {
	s.IDToEntity = IDToBaseEntity
	s.IDToEntityWithProcess = IDToEntityWithProcess

}

// InitProcesses инициализирует данные для процессов и запускает процессы.
// Информация берется из map[string]entities.Entity, где ключ = ID сущности, значение = сущность.
// map[string]entities.Entity создается из конфига устройств (приходит из другого сервиса).
func (s *SimEngine) InitProcesses() {
	for _, entity := range s.IDToEntityWithProcess {
		s.simulation.ProcessReflect(entity.GetProcessFunc())
	}
}

func (s *SimEngine) GetQueue() chan config.EventDTO {
	return s.eventsQueue
}

func (s *SimEngine) Run() error {
	// получаем новые ивенты и каждый новый ивент закидываем в HandleEvent
	defer close(s.eventsQueue)

	if s.simulation == nil {
		return errors.New("need simgo simulation")
	} else if s.Field == nil {
		return errors.New("need field")
	} else if s.eventsQueue == nil {
		return errors.New("need queue")
	}

	for event := range s.eventsQueue {
		s.HandleEvent(event)
	}

	return nil
}

// HandleEvent обрабатывает event по его entityID
func (s *SimEngine) HandleEvent(event config.EventDTO) {
	entity := s.IDToEntity[event.EntityID]
	receiversID := entity.GetReceiversID()

	for _, receiverID := range receiversID {
		s.eventsQueue <- config.EventDTO{ // может заблокироваться, можно добавить pending
			EntityID: receiverID,
			Cell:     entity.GetLocation(),
		}
	}

	if entityWithProcess, ok := s.IDToEntityWithProcess[event.EntityID]; ok {
		entityWithProcess.SendEvent(event)
	}
}
