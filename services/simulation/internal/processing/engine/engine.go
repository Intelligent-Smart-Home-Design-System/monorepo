package engine

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/fschuetz04/simgo"
)

const maxEventsBuffer = 100

// SimEngine реализует интерфефс Engine
type SimEngine struct {
	simulation *simgo.Simulation          // дискретная симуляция из simgo
	IDToEntity map[string]entities.Entity // ID сущности <-> структура сущности.
	Field *field.Field                    // Поле для симуляции
	eventsInChan chan api.EventInDTO      // Канал для входящих событий
	dtSim float64                         // шаг симуляционного времени, задаётся при создании
}

// NewSimEngine создает SimEngine
func NewSimEngine(dtSim float64) *SimEngine {
	return &SimEngine{
		simulation:    simgo.NewSimulation(),
		IDToEntity:    make(map[string]entities.Entity),
		eventsInChan:  make(chan api.EventInDTO, maxEventsBuffer),
		dtSim:        dtSim,
	}
}

// InitEntities инициализирует сущности и их зависимости.
func (s *SimEngine) InitEntities(
	IDToEntity map[string]entities.Entity,
	IDToDependencies map[string][]api.ActionDTO,
) {
	s.IDToEntity = IDToEntity

	for entityID, actions := range IDToDependencies {
		s.IDToEntity[entityID].SetReceivers(actions)
	}
}

// InitProcesses инициализирует данные для процессов и запускает процессы.
func (s *SimEngine) InitProcesses() {
	for _, entity := range s.IDToEntity {
		if entityWithProcess, ok := entity.(entities.EntityWithProcess); ok {
			s.simulation.ProcessReflect(entityWithProcess.GetProcessFunc())
		}
	}
}

// CheckCircleDependencies проверяет наличие циклических зависимостей среди сущностей.
// Возвращает true, если цикл найден.
func (s *SimEngine) CheckCircleDependencies() bool {
	color := make(map[string]int)

	for entityID := range s.IDToEntity {
		color[entityID] = 0
	}

	for entityID := range s.IDToEntity {
		if color[entityID] == 0 {
			if s.hasCycleDFS(entityID, color) {
				return true
			}
		}
	}

	return false
}

// hasCycleDFS выполняет DFS для обнаружения цикла.
// Возвращает true, если обнаружен цикл.
func (s *SimEngine) hasCycleDFS(entityID string, color map[string]int) bool {
	color[entityID] = 1

	receiversID := s.IDToEntity[entityID].GetReceiversID()
	for _, receiverID := range receiversID {
		if color[receiverID] == 1 {
			return true
		}

		if color[receiverID] == 0 {
			if s.hasCycleDFS(receiverID, color) {
				return true
			}
		}
	}

	color[entityID] = 2

	return false
}

// SetField устанавливает поле для симуляции.
func (s *SimEngine) SetField(simField *field.Field) {
	s.Field = simField
}

// GetInChan возвращает канал для входящих событий.
func (s *SimEngine) GetInChan() chan api.EventInDTO {
	return s.eventsInChan
}

// Run запускает симуляцию, обрабатывая события из канала eventsInChan.
// Если контекст отменен или канал закрыт, то симуляция завершается.
func (s *SimEngine) Run() error {
	for event := range s.eventsInChan {
		s.HandleEvent(event)
	}
	return nil
}

func (s *SimEngine) Step() {
	s.simulation.RunUntil(s.simulation.Now() + s.dtSim)
}

// CollectStep собирает состояния всех сущностей после тика.
// TODO: реализовать HasChanged/GetState на Entity для отдачи только изменившихся.
func (s *SimEngine) CollectStep(tick int) *api.SimulationStepPayload {
	var stateChanges []api.EntityDTO
 
	for id, entity := range s.IDToEntity {
		if entityWithState, ok := entity.(entities.EntityWithState); ok {
			if entityWithState.HasChanged() {
				stateChanges = append(stateChanges, api.EntityDTO{
					ID:   id,
					Info: entityWithState.GetState(),
				})
			}
		}
	}
 
	return &api.SimulationStepPayload{
		Tick:         tick,
		SimTime:      s.simulation.Now(),
		StateChanges: stateChanges,
	}
}

func (s *SimEngine) Stop() {
	close(s.eventsInChan)
}

// HandleEvent обрабатывает event по его entityID
func (s *SimEngine) HandleEvent(event api.EventInDTO) {
	entity := s.IDToEntity[event.EntityID]
	receiversID := entity.GetReceiversID()

	for _, receiverID := range receiversID {
		s.eventsInChan <- api.EventInDTO{
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

// UpdateField обновляет состояние ячейки на поле. Если координаты некорректные, то возвращает ошибку.
func (s *SimEngine) UpdateField(x, y int, cell field.Cell) error {
	if x < 0 || x > s.Field.Height {
		return ErrorFieldInvalidParameterX
	} else if y < 0 || y > s.Field.Width {
		return ErrorFieldInvalidParameterY
	}

	s.Field.Cells[x][y].Condition = cell.Condition

	return nil
}
