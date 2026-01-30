package engine

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/fschuetz04/simgo"
)

// SimEngine реализует интерфефс Engine
type SimEngine struct {
	simulation *simgo.Simulation // дискретная симуляция из simgo

	// IDToEntity хранит ключ = ID сущности, значение = структура сущности
	IDToEntity map[string]entities.Entity

	// Поле для симуляции
	Field *field.Field
}

// NewSimEngine создает SimEngine
func NewSimEngine(fieldDTO config.FieldDTO) *SimEngine {
	s := &SimEngine{
		simulation: simgo.NewSimulation(),
		IDToEntity: make(map[string]entities.Entity),
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
func (s *SimEngine) InitEntities(IDToEntity map[string]entities.Entity) {
	s.IDToEntity = IDToEntity
}

// InitProcesses инициализирует данные для процессов и запускает процессы.
// Информация берется из map[string]entities.Entity, где ключ = ID сущности, значение = сущность.
// map[string]entities.Entity создается из конфига устройств (приходит из другого сервиса).
func (s *SimEngine) InitProcesses() {
	for _, entity := range s.IDToEntity {
		s.simulation.ProcessReflect(entity.GetProcessFunc())
	}
}

func (s *SimEngine) Run() {
	// получаем новые ивенты и каждый новый ивент закидываем в HandleEvent
}

// HandleEvent обрабатывает event по его entityID
func (s *SimEngine) HandleEvent(entityID string) {
	//entityTriggers := s.IDToTriggers[entityID]
	//for _, entity := range entityTriggers {
	//	_, ok := s.devices[entityID]
	//	senderDelay := 0
	//	receiverDelay := 0
	//	if ok {
	//		// если девайс, то должны учесть задержку + бизнес логика
	//	}
	//	// устанавливаем Event (если нет) и вызываем trigger
	//	entity.SetEvent()
	//	//lampDelay := lamp.GetReactionDelay().Seconds()
	//	//lamp.trigger.TriggerDelayed(senderDelay + receiverDelay)
	//	entity.Trigger(senderDelay + receiverDelay)
	//}
}
