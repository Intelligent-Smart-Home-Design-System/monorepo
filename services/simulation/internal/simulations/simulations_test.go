package simulations

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
)

// =====Stubs=====
type stubEngine struct {
	inChan        chan api.EventDTO
	outChan       chan api.EventDTO
	stopCalled    bool
	stepCalled    bool
	handleErr     error
	stepErr       error
	stepStarted   chan struct{}
	stepRelease   chan struct{}
	runErr        error
	collectResult *api.SimulationStepPayload
}

func newStubEngine() *stubEngine {
	return &stubEngine{
		inChan:  make(chan api.EventDTO, 100),
		outChan: make(chan api.EventDTO, 100),
	}
}

func (s *stubEngine) InitEntities(IDToEntity map[string]entities.Entity, IDToDependencies map[string][]api.EdgeDTO) {
}

func (s *stubEngine) InitProcesses() {
}

func (s *stubEngine) CheckCircleDependencies() bool {
	return false
}

func (s *stubEngine) SetFloor(floor *api.Floor) {
}

func (s *stubEngine) GetInChan() chan api.EventDTO {
	return s.inChan
}

func (s *stubEngine) GetOutChan() chan api.EventDTO {
	return s.outChan
}

func (s *stubEngine) GetSimulation() interface{} {
	return nil
}

func (s *stubEngine) Run() error {
	return s.runErr
}

func (s *stubEngine) Step() error {
	s.stepCalled = true
	if s.stepStarted != nil {
		close(s.stepStarted)
	}
	if s.stepRelease != nil {
		<-s.stepRelease
	}
	return s.stepErr
}

func (s *stubEngine) CollectStep(tick int) *api.SimulationStepPayload {
	if s.collectResult != nil {
		return s.collectResult
	}

	return &api.SimulationStepPayload{Tick: tick}
}

func (s *stubEngine) Stop() {
	s.stopCalled = true
	close(s.inChan)
}

func (s *stubEngine) HandleEvent(event api.EventDTO) error {
	return s.handleErr
}

// =====Helper=====
func newTestSimulations() *Simulations {
	return NewSimulation()
}

func validStartPayload() api.SimulationStartPayload {
	floorObj := api.Floor{
		Meta: struct {
			Units string `json:"units"`
		}{
			Units: "meters",
		},
		Walls:   []api.Wall{},
		Doors:   []api.Door{},
		Windows: []api.Window{},
		Rooms:   []api.Room{},
	}

	rawApartment, _ := json.Marshal(floorObj)

	return api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: rawApartment,
		Devices:   []api.EntityDTO{},
		Scenarios: []api.ScenarioDTO{},
	}
}

// =====Tests=====
// Тест проверки инициализации
func TestNewSimulation(t *testing.T) {
	s := NewSimulation()

	if s == nil {
		t.Fatal("simulation is nil")
	}

	if s.sessions == nil {
		t.Fatal("sessions not initialized")
	}
}

// Тест проверки функции Start()
func TestStart(t *testing.T) {
	reqID := "sim1"
	payload := validStartPayload()

	s := newTestSimulations()

	err := s.Start(reqID, payload)
	if err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}

	s.mu.RLock()
	_, ok := s.sessions[reqID]
	s.mu.RUnlock()

	if !ok {
		t.Errorf("engine not registered for reqID %q", reqID)
	}
}

// TestStart_RejectsExistingEngine проверяет, что другая WebSocket-сессия не может заменить работающий engine.
func TestStart_RejectsExistingEngine(t *testing.T) {
	s := newTestSimulations()
	previous := newStubEngine()
	s.sessions["sim1"] = &simulationSession{engine: previous}

	if err := s.Start("sim1", validStartPayload()); err == nil {
		t.Fatal("Start() error = nil, want duplicate simulation error")
	}
	if previous.stopCalled {
		t.Fatal("existing engine was stopped by duplicate Start()")
	}
	if s.sessions["sim1"].engine != previous {
		t.Fatal("existing engine was replaced by duplicate Start()")
	}
}

// Тест проверки функции Tick() когда симуляция не найдена
func TestTick_NotFound(t *testing.T) {
	s := newTestSimulations()

	_, err := s.Tick("nonexistent", api.SimulationTickPayload{Tick: 1})
	if err == nil {
		t.Fatal("expected error for unknown reqID, got nil")
	}
}

// Тест проверки функции Tick() с корректным reqID
func TestTick_Success(t *testing.T) {
	s := newTestSimulations()

	stub := newStubEngine()
	stub.collectResult = &api.SimulationStepPayload{Tick: 5, SimTime: 5.0}
	s.sessions["sim1"] = &simulationSession{engine: stub}

	inputPayload, _ := json.Marshal(map[string]bool{"turn_on": true})
	tickPayload := api.SimulationTickPayload{
		Tick:   5,
		Inputs: []api.EventDTO{{EntityID: "lamp_1", Payload: inputPayload}},
	}

	result, err := s.Tick("sim1", tickPayload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tick != 5 {
		t.Errorf("tick = %v, want 5", result.Tick)
	}

	if !stub.stepCalled {
		t.Error("Step() was not called")
	}
}

func TestTick_ReturnsInputError(t *testing.T) {
	s := newTestSimulations()
	stub := newStubEngine()
	stub.handleErr = errors.New("invalid device payload")
	s.sessions["sim1"] = &simulationSession{engine: stub}

	_, err := s.Tick("sim1", api.SimulationTickPayload{
		Tick:   1,
		Inputs: []api.EventDTO{{EntityID: "device_1", Payload: json.RawMessage(`{}`)}},
	})
	if err == nil || err.Error() != "input 0: invalid device payload" {
		t.Fatalf("Tick() error = %v, want input error", err)
	}
	if stub.stepCalled {
		t.Fatal("Step() was called after an invalid input")
	}
}

func TestTick_DifferentSessionsRunIndependently(t *testing.T) {
	s := newTestSimulations()
	slow := newStubEngine()
	slow.stepStarted = make(chan struct{})
	slow.stepRelease = make(chan struct{})
	fast := newStubEngine()
	s.sessions["slow"] = &simulationSession{engine: slow}
	s.sessions["fast"] = &simulationSession{engine: fast}

	slowDone := make(chan error, 1)
	go func() {
		_, err := s.Tick("slow", api.SimulationTickPayload{Tick: 1})
		slowDone <- err
	}()
	<-slow.stepStarted

	fastDone := make(chan error, 1)
	go func() {
		_, err := s.Tick("fast", api.SimulationTickPayload{Tick: 1})
		fastDone <- err
	}()

	select {
	case err := <-fastDone:
		if err != nil {
			t.Fatalf("fast Tick() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("fast session was blocked by another session tick")
	}

	close(slow.stepRelease)
	if err := <-slowDone; err != nil {
		t.Fatalf("slow Tick() error = %v", err)
	}
}

// Тест проверки функции Stop() когда симуляция не найдена
func TestStop_NotFound(t *testing.T) {
	s := newTestSimulations()

	err := s.Stop("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown reqID, got nil")
	}
}

// Тест проверки функции Stop() с корректным reqID
func TestStop_Success(t *testing.T) {
	s := newTestSimulations()

	stub := newStubEngine()
	s.sessions["sim1"] = &simulationSession{engine: stub}

	err := s.Stop("sim1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !stub.stopCalled {
		t.Error("Stop() was not called on engine")
	}

	s.mu.RLock()
	_, ok := s.sessions["sim1"]
	s.mu.RUnlock()

	if ok {
		t.Error("engine was not removed after Stop()")
	}
}

func TestStart_CircleDependencies(t *testing.T) {
	s := newTestSimulations()

	// Сценарий с циклом: lamp_1 -> lamp_2 -> lamp_1
	payload := validStartPayload()
	payload.Devices = []api.EntityDTO{
		{ID: "lamp_1", Type: "lamp_switcher", Info: json.RawMessage(`{"id":"lamp_1","turn_on":false,"delay":0}`)},
		{ID: "lamp_2", Type: "lamp_switcher", Info: json.RawMessage(`{"id":"lamp_2","turn_on":false,"delay":0}`)},
	}
	payload.Scenarios = []api.ScenarioDTO{
		{EntityID: "lamp_1", Edges: []api.EdgeDTO{{ToID: "lamp_2"}}},
		{EntityID: "lamp_2", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
	}

	err := s.Start("sim_cycle", payload)
	if err == nil {
		t.Fatal("expected error for circular dependencies, got nil")
	}

	if err.Error() != "circle dependencies detected" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestStart_RejectsMissingScenarioEntities(t *testing.T) {
	tests := []struct {
		name      string
		scenarios []api.ScenarioDTO
		wantError string
	}{
		{
			name:      "missing source",
			scenarios: []api.ScenarioDTO{{EntityID: "missing", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}}},
			wantError: `scenario source entity "missing" does not exist`,
		},
		{
			name:      "missing target",
			scenarios: []api.ScenarioDTO{{EntityID: "lamp_1", Edges: []api.EdgeDTO{{ToID: "missing"}}}},
			wantError: `scenario target entity "missing" does not exist`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validStartPayload()
			payload.Devices = []api.EntityDTO{{
				ID:   "lamp_1",
				Type: "lamp",
				Info: json.RawMessage(`{"id":"lamp_1","turn_on":false,"delay":0}`),
			}}
			payload.Scenarios = tt.scenarios

			err := newTestSimulations().Start("invalid-dependency", payload)
			if err == nil || err.Error() != tt.wantError {
				t.Fatalf("Start() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}
