package engine

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/devices"
	"github.com/fschuetz04/simgo"
)

type SimEngine struct {
	simulation *simgo.Simulation
	started    bool

	// IDToEntity хранит ключ = ID сущности, значение = структура сущности
	IDToEntity map[string]entities.Entity

	// IDToTriggers хранит ключ = ID сущности, которая тригерит и значение = слайс сущностей, которые тригерятся
	IDToTriggers map[string][]entities.Entity

	// IDToInData хранит ID сущности и структуру для входных данных этой сущности
	IDToInData map[string]domain.InData

	// IDToOutData хранит ID сущности и структуру для выходных данных этой сущности
	IDToOutData map[string]domain.OutData
}

// NewSimEngine создает SimEngine
func NewSimEngine() *SimEngine {
	return &SimEngine{
		simulation:   simgo.NewSimulation(),
		started:      false,
		IDToTriggers: make(map[string][]entities.Entity),
	}
}

// InitEntities создает сущности для симуляции из мапы с конфигом.
// IDToEntityType хранит ключ = ID сущности, значение = конфиг сущности.
func (s *SimEngine) InitEntities(IDToEntityType map[string]config.EntityConfig) {
	for _, cfg := range IDToEntityType {
		switch cfg.Type {
		case "lamp":
			s.IDToEntity[cfg.ID] = devices.NewLamp(cfg.ID, float64(cfg.Delay))
		}
	}
}

// InitDependencies инициализирует зависимости между сущностями
func (s *SimEngine) InitDependencies(dependencies map[string][]string) {
	for sender, receivers := range dependencies {
		for _, receiver := range receivers {
			s.IDToTriggers[sender] = append(s.IDToTriggers[sender], s.IDToEntity[receiver])
		}
	}
}

// InitProcesses инициализирует данные для процессов и запускает процессы.
// Информация берется из map[string]entities.Entity, где ключ = ID сущности, значение = сущность.
// map[string]entities.Entity создается из конфига устройств (приходит из другого сервиса).
func (s *SimEngine) InitProcesses(simEntities map[string]entities.Entity) {
	for _, simEntity := range simEntities {
		entityID := simEntity.GetID()
		s.IDToInData[entityID] = simEntity.GetInDataStruct()
		s.IDToOutData[entityID] = simEntity.GetOutDataStruct()

		s.simulation.ProcessReflect(simEntity.GetProcessFunc(), s.IDToInData[entityID], s.IDToOutData[entityID])
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
