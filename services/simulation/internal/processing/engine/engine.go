package engine

import (
	"context"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/fschuetz04/simgo"
)

const (
	maxEventsBuffer = 100
	simStep         = 1.0
)

// SimEngine реализует интерфефс Engine
type SimEngine struct {
	simulation *simgo.Simulation // дискретная симуляция из simgo

	IDToEntity map[string]entities.Entity // ID сущности <-> структура сущности.

	Field *field.Field // Поле для симуляции

	eventsInChan chan api.EventInDTO // Канал для входящих событий

	eventsOutChan chan api.EventOutDTO // Канал для новых событий
}

// NewSimEngine создает SimEngine
func NewSimEngine() *SimEngine {
	return &SimEngine{
		simulation:    simgo.NewSimulation(),
		IDToEntity:    make(map[string]entities.Entity),
		eventsInChan:  make(chan api.EventInDTO, maxEventsBuffer),
		eventsOutChan: make(chan api.EventOutDTO, maxEventsBuffer),
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

// GetOutChan возвращает канал для исходящих событий.
func (s *SimEngine) GetOutChan() chan api.EventOutDTO {
	return s.eventsOutChan
}

// Run запускает симуляцию, обрабатывая события из канала eventsInChan.
// Если контекст отменен или канал закрыт, то симуляция завершается.
func (s *SimEngine) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-s.eventsInChan:
			if !ok {
				return nil
			}

			eventType := strings.Split(event.EntityID, "_")[0]
			if eventType == "step" {
				s.simulation.RunUntil(s.simulation.Now() + simStep) // шаг симуляции (можно делать каждый lockstep)
			} else {
				s.HandleEvent(event)

				s.simulation.RunUntil(s.simulation.Now() + simStep)
			}
		}
	}
}

// HandleEvent обрабатывает event по его entityID
func (s *SimEngine) HandleEvent(event api.EventInDTO) {
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

// GetSimulation возвращает симуляцию для взаимодействия сущностей с движком
func (s *SimEngine) GetSimulation() *simgo.Simulation {
	return s.simulation
}
