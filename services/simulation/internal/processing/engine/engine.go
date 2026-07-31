package engine

import (
	"fmt"
	"math"

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
	dependencies  map[string][]api.EdgeDTO   // постоянные сценарные связи sourceID <-> edges
	eventsInChan  chan api.EventDTO          // Канал для входящих событий
	eventsOutChan chan api.EventDTO          // Канал для выходящих событий
	triggeredChan chan api.EdgeDTO           // Сработавшие постоянные связи текущего шага
	dtSim         float64                    // шаг симуляционного времени, задаётся при создании
	Floor         *api.Floor                 // Поле для симуляции
	stepErr       error                      // первая ошибка обработки события текущего шага
}

// NewSimEngine создает SimEngine
func NewSimEngine(dtSim float64) *SimEngine {
	return &SimEngine{
		simulation:    simgo.NewSimulation(),
		IDToEntity:    make(map[string]entities.Entity),
		roomObservers: make(map[string][]string),
		dependencies:  make(map[string][]api.EdgeDTO),
		eventsInChan:  make(chan api.EventDTO, maxEventsBuffer),
		eventsOutChan: make(chan api.EventDTO, maxEventsBuffer),
		triggeredChan: make(chan api.EdgeDTO, maxEventsBuffer),
		dtSim:         dtSim,
	}
}

// InitEntities инициализирует сущности и их зависимости.
func (s *SimEngine) InitEntities(
	IDToEntity map[string]entities.Entity,
	IDToDependencies map[string][]api.EdgeDTO,
) {
	s.IDToEntity = IDToEntity
	s.dependencies = IDToDependencies

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

// TriggerReceivers передаёт payload всем постоянным получателям sourceID и записывает реально сработавшие связи.
func (s *SimEngine) TriggerReceivers(sourceID string, payload []byte) {
	for _, edge := range s.dependencies[sourceID] {
		s.eventsInChan <- api.EventDTO{
			EntityID: edge.ToID,
			Payload:  payload,
		}
		s.triggeredChan <- api.EdgeDTO{
			FromID: sourceID,
			ToID:   edge.ToID,
			Action: edge.Action,
			Data:   edge.Data,
		}
	}
}

// GetSimulation возвращает дикретную симуляцию simgo.
func (s *SimEngine) GetSimulation() *simgo.Simulation {
	return s.simulation
}

// InitStep выполняет события запуска процессов в текущем модельном времени.
func (s *SimEngine) InitStep() {
	s.runCurrentTime()
}

// Step выполняет шаг симуляции и возвращает первую ошибку обработки события.
func (s *SimEngine) Step() error {
	s.stepErr = nil
	targetTime := s.simulation.Now() + s.dtSim

	// Пользовательские события и порождённые ими мгновенные цепочки должны
	// завершиться до вызова периодической логики сущностей.
	s.processCurrentEvents()
	if s.stepErr != nil {
		return s.stepErr
	}

	for _, entity := range s.IDToEntity {
		if t, ok := entity.(entities.Tickable); ok {
			t.OnTick()
		}
	}

	// OnTick может породить входы для observers и связанных устройств.
	s.processCurrentEvents()
	if s.stepErr != nil {
		return s.stepErr
	}

	s.simulation.RunUntil(targetTime)
	return s.stepErr
}

// processCurrentEvents передаёт входы сущностям и выполняет все события,
// запланированные на текущее модельное время, включая мгновенные цепочки.
func (s *SimEngine) processCurrentEvents() {
	s.drainInChan()

	for s.stepErr == nil {
		s.runCurrentTime()

		// Обработка сущности может породить входы для связанных сущностей.
		// Передаём их в Store и повторяем текущее модельное время.
		if s.drainInChan() == 0 {
			return
		}
	}
}

// runCurrentTime выполняет события с timestamp, равным текущему времени.
// RunUntil использует строгое сравнение, поэтому target должен быть больше now.
func (s *SimEngine) runCurrentTime() {
	now := s.simulation.Now()
	s.simulation.RunUntil(math.Nextafter(now, math.Inf(1)))
}

// DrainInChan читает все доступные события из канала
func (s *SimEngine) DrainInChan() {
	s.drainInChan()
}

// drainInChan передаёт доступные входные события сущностям и возвращает их количество.
func (s *SimEngine) drainInChan() int {
	drained := 0
	for {
		select {
		case event, ok := <-s.eventsInChan:
			if !ok {
				return drained
			}
			drained++

			if err := s.HandleEvent(event); err != nil && s.stepErr == nil {
				s.stepErr = err
			}
		default:
			return drained
		}
	}
}

// CollectStep собирает обновления от всех сущностей после тика.
func (s *SimEngine) CollectStep(tick int) *api.SimulationStepPayload {
	changes := make([]api.EventDTO, 0)
	triggeredEdges := make([]api.EdgeDTO, 0)
	seenEdges := make(map[[3]string]struct{})
	humans := make([]api.EntityDTO, 0)

	for {
		select {
		case edge := <-s.triggeredChan:
			key := [3]string{edge.FromID, edge.ToID, edge.Action}
			if _, exists := seenEdges[key]; exists {
				continue
			}
			seenEdges[key] = struct{}{}
			triggeredEdges = append(triggeredEdges, edge)
		default:
			goto collectChanges
		}
	}

collectChanges:
	for {
		select {
		case event := <-s.eventsOutChan:
			changes = append(changes, event)
		default:
			return &api.SimulationStepPayload{
				Tick:           tick,
				SimTime:        s.simulation.Now(),
				StateChanges:   changes,
				TriggeredEdges: triggeredEdges,
				Humans:         humans,
			}
		}
	}
}

// Stop останавливает симуляцию, закрывая канал входящих событий.
func (s *SimEngine) Stop() {
	close(s.eventsInChan)
}

// HandleEvent обрабатывает event по его entityID или возвращает ошибку контракта.
func (s *SimEngine) HandleEvent(event api.EventDTO) error {
	entity, exists := s.IDToEntity[event.EntityID]
	if !exists {
		return fmt.Errorf("entity %q does not exist", event.EntityID)
	}

	entityWithProcess, ok := entity.(entities.EntityWithProcess)
	if !ok {
		return fmt.Errorf("entity %q cannot process input events", event.EntityID)
	}
	if err := entityWithProcess.HandleInDTO(event.Payload); err != nil {
		return fmt.Errorf("entity %q rejected input: %w", event.EntityID, err)
	}

	return nil
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
