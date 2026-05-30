package simulations

import (
	"encoding/json"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
)

// =====Stubs=====
type stubEngine struct {
	inChan        chan api.EventInDTO
	outChan       chan api.EventOutDTO
	stopCalled    bool
	stepCalled    bool
	runErr        error
	collectResult *api.SimulationStepPayload
}

func newStubEngine() *stubEngine {
	return &stubEngine{
		inChan:  make(chan api.EventInDTO, 100),
		outChan: make(chan api.EventOutDTO, 100),
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

func (s *stubEngine) GetInChan() chan api.EventInDTO {
	return s.inChan
}

func (s *stubEngine) GetOutChan() chan api.EventOutDTO {
	return s.outChan
}

func (s *stubEngine) GetSimulation() interface{} {
	return nil
}

func (s *stubEngine) Run() error {
	return s.runErr
}

func (s *stubEngine) Step() {
	s.stepCalled = true
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

func (s *stubEngine) HandleEvent(event api.EventInDTO) {
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
	if s.IDToEngine == nil {
		t.Fatal("IDToEngine not initialized")
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
	_, ok := s.IDToEngine[reqID]
	s.mu.RUnlock()

	if !ok {
		t.Errorf("engine not registered for reqID %q", reqID)
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
	s.IDToEngine["sim1"] = stub

	inputPayload, _ := json.Marshal(map[string]bool{"turn_on": true})
	tickPayload := api.SimulationTickPayload{
		Tick:   5,
		Inputs: []api.EventInDTO{{EntityID: "lamp_1", Payload: inputPayload}},
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
	s.IDToEngine["sim1"] = stub

	err := s.Stop("sim1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.stopCalled {
		t.Error("Stop() was not called on engine")
	}

	s.mu.RLock()
	_, ok := s.IDToEngine["sim1"]
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
