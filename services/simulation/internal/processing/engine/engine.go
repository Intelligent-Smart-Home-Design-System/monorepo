package engine

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/fschuetz04/simgo"
)

const maxEventsBuffer = 100

// SimEngine реализует интерфефс Engine
type SimEngine struct {
	simulation    *simgo.Simulation          // дискретная симуляция из simgo
	IDToEntity    map[string]entities.Entity // ID сущности <-> структура сущности.
	roomObservers map[string][]string        // roomID <-> []entityID (entity с логикой entities.Observer)
	eventsInChan  chan api.EventDTO          // Канал для входящих событий
	eventsOutChan chan api.EventDTO          // Канал для выходящих событий
	dtSim         float64                    // шаг симуляционного времени, задаётся при создании
	Floor         *api.Floor                 // Поле для симуляции
}

// NewSimEngine создает SimEngine
func NewSimEngine(dtSim float64) *SimEngine {
	return &SimEngine{
		simulation:    simgo.NewSimulation(),
		IDToEntity:    make(map[string]entities.Entity),
		roomObservers: make(map[string][]string),
		eventsInChan:  make(chan api.EventDTO, maxEventsBuffer),
		eventsOutChan: make(chan api.EventDTO, maxEventsBuffer),
		dtSim:         dtSim,
	}
}

// InitEntities инициализирует сущности и их зависимости.
func (s *SimEngine) InitEntities(
	IDToEntity map[string]entities.Entity,
	IDToDependencies map[string][]api.EdgeDTO,
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

		if observer, ok := entity.(entities.Observer); ok {
			x, y := observer.GetPosition()
			for _, room := range s.Floor.Rooms {
				if field.PointInRoom(x, y, room) {
					s.roomObservers[room.ID] = append(s.roomObservers[room.ID], observer.GetID())
					break
				}
			}
		}
	}
}

// GetRoomObservers возвращает ID observers в комнате.
func (s *SimEngine) GetRoomObservers(roomID string) []string {
	return s.roomObservers[roomID]
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

// SetFloor устанавливает поле для симуляции.
func (s *SimEngine) SetFloor(floor *api.Floor) {
	s.Floor = floor
}

// GetInChan возвращает канал для входящих событий.
func (s *SimEngine) GetInChan() chan api.EventDTO {
	return s.eventsInChan
}

// GetOutChan возвращает канал для выходящих событий.
func (s *SimEngine) GetOutChan() chan api.EventDTO {
	return s.eventsOutChan
}

// GetSimulation возвращает дикретную симуляцию simgo.
func (s *SimEngine) GetSimulation() *simgo.Simulation {
	return s.simulation
}

// InitStep запускает симуляцию до 0, чтобы инициализировать процессы.
func (s *SimEngine) InitStep() {
	s.simulation.RunUntil(0)
}

// Step выполняет шаг симуляции.
func (s *SimEngine) Step() {
	targetTime := s.simulation.Now() + s.dtSim

	for _, entity := range s.IDToEntity {
		if t, ok := entity.(entities.Tickable); ok {
			tickPayload, _ := json.Marshal(struct {
				Tick bool `json:"tick"`
			}{Tick: true})
			s.eventsInChan <- api.EventInDTO{
				EntityID: t.GetID(),
				Payload:  tickPayload,
			}
		}
	}

	s.DrainInChan()

	s.simulation.RunUntil(targetTime)
}

// DrainInChan читает все доступные события из канала
func (s *SimEngine) DrainInChan() {
	for {
		select {
		case event, ok := <-s.eventsInChan:
			if !ok {
				return
			}
			s.HandleEvent(event)
		default:
			return
		}
	}
}

// CollectStep собирает обновления от всех сущностей после тика.
func (s *SimEngine) CollectStep(tick int) *api.SimulationStepPayload {
	changes := make([]api.EventDTO, 0)

	for {
		select {
		case event := <-s.eventsOutChan:
			changes = append(changes, event)
		default:
			return &api.SimulationStepPayload{
				Tick:         tick,
				SimTime:      s.simulation.Now(),
				StateChanges: changes,
			}
		}
	}
}

// Stop останавливает симуляцию, закрывая канал входящих событий.
func (s *SimEngine) Stop() {
	close(s.eventsInChan)
}

// HandleEvent обрабатывает event по его entityID
func (s *SimEngine) HandleEvent(event api.EventDTO) {
	if entityWithProcess, ok := s.IDToEntity[event.EntityID].(entities.EntityWithProcess); ok {
		err := entityWithProcess.HandleInDTO(event.Payload)
		if err != nil {
			return
		}
	}
}

// NotifyObservers отправляет payload всем observer в комнате roomID, которые слушают kind событий.
func (s *SimEngine) NotifyObservers(roomID string, kind string, payload []byte) {
	for _, observerID := range s.roomObservers[roomID] {
		entity := s.IDToEntity[observerID]

		observer, ok := entity.(entities.Observer)
		if !ok {
			continue
		}

		for _, k := range observer.GetObservedKinds() {
			if k == kind {
				s.eventsInChan <- api.EventDTO{
					EntityID: observerID,
					Payload:  payload,
				}

				break
			}
		}
	}
}

// GetFloor возвращает поле для симуляции.
func (s *SimEngine) GetFloor() *api.Floor {
	return s.Floor
}

func (s *SimEngine) GetEntity(id string) entities.Entity {
    return s.IDToEntity[id]
}
