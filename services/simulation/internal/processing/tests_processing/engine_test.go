package tests_processing

import (
	"errors"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// =====Stubs=====
type stubEntity struct {
	id        string
	receivers []string
	setCalled bool
}

func (s *stubEntity) GetID() string {
	return s.id
}

func (s *stubEntity) GetReceiversID() []string {
	return s.receivers
}

func (s *stubEntity) SetReceivers(actions []api.EdgeDTO) {
	s.setCalled = true

	var ids []string
	for _, a := range actions {
		ids = append(ids, a.ToID)
	}

	s.receivers = ids
}

type stubEntityWithProcess struct {
	stubEntity
	handleErr   error
	handleCalls int
}

func (s *stubEntityWithProcess) GetProcessFunc() func(process simgo.Process) {
	return func(process simgo.Process) {}
}

func (s *stubEntityWithProcess) HandleInDTO(dto []byte) error {
	s.handleCalls++
	return s.handleErr
}

func (s *stubEntityWithProcess) HandleOutDTO(dto []byte) {
}

func (s *stubEntityWithProcess) Process(process simgo.Process) {
}

func (s *stubEntityWithProcess) GetOutCh() chan []byte {
	return make(chan []byte)
}

type orderedTickEntity struct {
	stubEntity
	store             *simgo.Store[[]byte]
	eventHandled      bool
	eventSeenByOnTick bool
}

func newOrderedTickEntity(id string, simulation *simgo.Simulation) *orderedTickEntity {
	return &orderedTickEntity{
		stubEntity: stubEntity{id: id},
		store:      simgo.NewStore[[]byte](simulation),
	}
}

func (s *orderedTickEntity) GetProcessFunc() func(process simgo.Process) {
	return s.Process
}

func (s *orderedTickEntity) HandleInDTO(dto []byte) error {
	s.store.Put(dto)
	return nil
}

func (s *orderedTickEntity) HandleOutDTO([]byte) {}

func (s *orderedTickEntity) Process(process simgo.Process) {
	for {
		element := s.store.Get()
		process.Wait(element.Event)
		s.eventHandled = true
	}
}

func (s *orderedTickEntity) OnTick() {
	s.eventSeenByOnTick = s.eventHandled
}

// =====Tests=====
// Тест проверки функции CheckCircleDependencies()
func TestCheckCircleDependencies(t *testing.T) {
	tests := []struct {
		name     string
		entities map[string]entities.Entity
		want     bool
	}{
		{
			name: "no cycle",
			entities: map[string]entities.Entity{
				"a": &stubEntity{id: "a", receivers: []string{"b"}},
				"b": &stubEntity{id: "b", receivers: []string{}},
			},
			want: false,
		},
		{
			name: "simple cycle",
			entities: map[string]entities.Entity{
				"a": &stubEntity{id: "a", receivers: []string{"b"}},
				"b": &stubEntity{id: "b", receivers: []string{"a"}},
			},
			want: true,
		},
		{
			name: "self cycle",
			entities: map[string]entities.Entity{
				"a": &stubEntity{id: "a", receivers: []string{"a"}},
			},
			want: true,
		},
		{
			name:     "empty entities",
			entities: map[string]entities.Entity{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := engine.NewSimEngine(1.0)
			e.IDToEntity = tt.entities

			got := e.CheckCircleDependencies()
			if got != tt.want {
				t.Errorf("CheckCircleDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Тест проверки функции HandleEvent()
func TestHandleEvent(t *testing.T) {
	tests := []struct {
		name        string
		entity      *stubEntityWithProcess
		expectCalls int
		expectEnq   bool
	}{
		{
			name: "handle success",
			entity: &stubEntityWithProcess{
				stubEntity: stubEntity{id: "a", receivers: []string{"b"}},
			},
			expectCalls: 1,
			expectEnq:   true,
		},
		{
			name: "handle returns error",
			entity: &stubEntityWithProcess{
				stubEntity: stubEntity{id: "a", receivers: []string{}},
				handleErr:  errors.New("fail"),
			},
			expectCalls: 1,
			expectEnq:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := engine.NewSimEngine(1.0)
			e.IDToEntity["a"] = tt.entity
			e.IDToEntity["b"] = &stubEntity{id: "b"}
			event := api.EventDTO{EntityID: "a"}

			err := e.HandleEvent(event)

			if tt.entity.handleCalls != tt.expectCalls {
				t.Errorf("handle calls = %v, want %v", tt.entity.handleCalls, tt.expectCalls)
			}
			if (err != nil) != (tt.entity.handleErr != nil) {
				t.Errorf("HandleEvent() error = %v, handler error = %v", err, tt.entity.handleErr)
			}
		})
	}
}

func TestHandleEvent_RejectsUnknownEntity(t *testing.T) {
	e := engine.NewSimEngine(1.0)

	err := e.HandleEvent(api.EventDTO{EntityID: "missing", Payload: []byte(`{}`)})
	if err == nil {
		t.Fatal("HandleEvent() error = nil, want unknown entity error")
	}
}

// TestTriggerReceiversCollectsUniqueEdges проверяет маршрутизацию и однократный вывод постоянной связи за tick.
func TestTriggerReceiversCollectsUniqueEdges(t *testing.T) {
	e := engine.NewSimEngine(1.0)
	e.InitEntities(
		map[string]entities.Entity{
			"sensor_1": &stubEntity{id: "sensor_1"},
			"lamp_1":   &stubEntity{id: "lamp_1"},
		},
		map[string][]api.EdgeDTO{
			"sensor_1": {
				{ToID: "lamp_1", Action: "trigger", Data: []interface{}{"night"}},
			},
		},
	)

	payload := []byte(`{"turn_on":true}`)
	e.TriggerReceivers("sensor_1", payload)
	e.TriggerReceivers("sensor_1", payload)

	step := e.CollectStep(7)
	if len(step.TriggeredEdges) != 1 {
		t.Fatalf("expected one unique triggered edge, got %d", len(step.TriggeredEdges))
	}

	edge := step.TriggeredEdges[0]
	if edge.FromID != "sensor_1" || edge.ToID != "lamp_1" || edge.Action != "trigger" {
		t.Fatalf("unexpected triggered edge: %+v", edge)
	}
	if len(edge.Data) != 1 || edge.Data[0] != "night" {
		t.Fatalf("triggered edge data was not preserved: %+v", edge.Data)
	}

	nextStep := e.CollectStep(8)
	if len(nextStep.TriggeredEdges) != 0 {
		t.Fatalf("triggered edges must be cleared after collection, got %+v", nextStep.TriggeredEdges)
	}
}

// TestStepProcessesInputsBeforeOnTick проверяет, что события текущего шага
// применяются сущностью до однократного вызова её периодической логики.
func TestStepProcessesInputsBeforeOnTick(t *testing.T) {
	e := engine.NewSimEngine(1.0)
	entity := newOrderedTickEntity("ordered", e.GetSimulation())
	e.InitEntities(map[string]entities.Entity{"ordered": entity}, nil)
	e.InitProcesses()
	e.InitStep()
	if err := e.HandleEvent(api.EventDTO{EntityID: "ordered", Payload: []byte(`{}`)}); err != nil {
		t.Fatalf("HandleEvent() error = %v", err)
	}

	if err := e.Step(); err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if !entity.eventSeenByOnTick {
		t.Fatal("OnTick observed state before the input event was processed")
	}
}

// Тест корректного поведения функции Run() при закрытии канала
func TestRun_ChannelClosed(t *testing.T) {
	t.Run("channel closed", func(t *testing.T) {
		e := engine.NewSimEngine(1.0)
		close(e.GetInChan())

		if err := e.Step(); err != nil {
			t.Fatalf("Step() error = %v", err)
		}
		t.Logf("Closed channel correct working")
	})
}
