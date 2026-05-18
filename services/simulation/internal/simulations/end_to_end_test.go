package simulations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/client/ws"
	"github.com/gorilla/websocket"
)

// ===== Helper =====
func dialSim(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dialSim: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

func sendMsg(t *testing.T, conn *websocket.Conn, msg api.Message) {
	t.Helper()

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("sendMsg: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("sendMsg: %v", err)
	}
}

func recvMsg(t *testing.T, conn *websocket.Conn) api.Message {
	t.Helper()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("recvMsg: %v", err)
	}

	var msg api.Message

	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("recvMsg: %v", err)
	}

	return msg
}

func recvStep(t *testing.T, conn *websocket.Conn) api.SimulationStepPayload {
	t.Helper()

	msg := recvMsg(t, conn)

	if msg.Type != "simulation:step" {
		t.Fatalf("expected simulation:step, got %q", msg.Type)
	}

	var step api.SimulationStepPayload

	if err := json.Unmarshal(msg.Payload, &step); err != nil {
		t.Fatalf("recvStep: %v", err)
	}

	return step
}

func newSimServer(t *testing.T) *httptest.Server {
	t.Helper()
	simService := NewSimulation()
	manager := ws.NewManager(simService)
	server := httptest.NewServer(http.HandlerFunc(manager.ServeWS))
	t.Cleanup(server.Close)
	return server
}

func startSim(t *testing.T, conn *websocket.Conn, reqID string, payload api.SimulationStartPayload) {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("startSim: %v", err)
	}

	sendMsg(t, conn, api.Message{Type: "simulation:start", Ts: time.Now(), ReqID: reqID, Payload: raw})
	msg := recvMsg(t, conn)
	if msg.Type != "simulation:started" {
		t.Fatalf("startSim: expected simulation:started, got %q", msg.Type)
	}
}

func tick(t *testing.T, conn *websocket.Conn, reqID string, tickN int, inputs []api.EventInDTO) api.SimulationStepPayload {
	t.Helper()

	raw, err := json.Marshal(api.SimulationTickPayload{Tick: tickN, Inputs: inputs})
	if err != nil {
		t.Fatalf("tick: %v", err)
	}

	sendMsg(t, conn, api.Message{
		Type:    "simulation:tick",
		Ts:      time.Now(),
		ReqID:   reqID,
		Payload: raw,
	})

	return recvStep(t, conn)
}

func inputEvent(t *testing.T, entityID string, turnOn bool) api.EventInDTO {
	t.Helper()

	payload, err := json.Marshal(map[string]bool{"turn_on": turnOn})
	if err != nil {
		t.Fatalf("inputEvent: %v", err)
	}

	return api.EventInDTO{
		EntityID: entityID,
		Payload:  payload,
	}
}

// statesFrom собирает последовательность turn_on из StateChanges для конкретного entityID.
func statesFrom(steps []api.SimulationStepPayload, entityID string) []bool {
	var result []bool
	for _, step := range steps {
		for _, change := range step.StateChanges {
			if change.EntityID != entityID {
				continue
			}

			var out struct {TurnOn bool `json:"turn_on"`}
			_ = json.Unmarshal(change.Payload, &out)
			result = append(result, out.TurnOn)
		}
	}

	return result
}

// lastStateOf возвращает последнее значение turn_on для entityID из всех шагов.
func lastStateOf(steps []api.SimulationStepPayload, entityID string) (bool, bool) {
	states := statesFrom(steps, entityID)

	if len(states) == 0 {
		return false, false
	}

	return states[len(states)-1], true
}

// ===== Tests =====
// Тест проверки корректности работы программы в стандартном случае.
func TestSimulation_Default(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-default"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Apartment: api.FieldDTO{Width: 2, Height: 2, Cells: [][]*api.CellDTO{}},
		Devices: []api.EntityDTO{
			{ID: "lampSwitcher_1", Type: "lampSwitcher", Info: json.RawMessage(`{"id":"lampSwitcher_1", "delay":1.0}`)},
			{ID: "lampSwitcher_2", Type: "lampSwitcher", Info: json.RawMessage(`{"id":"lampSwitcher_2", "delay":0.3}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1", "delay":0.5}`)},
			{ID: "lamp_2", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_2", "delay":1.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "lampSwitcher_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
			{EntityID: "lampSwitcher_2", Edges: []api.EdgeDTO{{ToID: "lamp_1"}, {ToID: "lamp_2"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "lampSwitcher_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "lampSwitcher_2", true)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventInDTO{inputEvent(t, "lampSwitcher_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 4, nil))
	steps = append(steps, tick(t, conn, reqID, 5, nil))
	steps = append(steps, tick(t, conn, reqID, 6, nil))

	total := 0

	for _, s := range steps {
		total += len(s.StateChanges)
	}
	if total != 7 {
		t.Fatalf("expected 7 total state changes, got %d", total)
	}
}

// Тест проверки корректности работы программы при вмешательстве пользователя.
func TestWS_Simulation_UserIntervention(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-intervention"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 10.0,
		Apartment: api.FieldDTO{Width: 2, Height: 2, Cells: [][]*api.CellDTO{}},
		Devices: []api.EntityDTO{
			{ID: "lampSwitcher_1", Type: "lampSwitcher", Info: json.RawMessage(`{"id":"lampSwitcher_1","delay":0.0}`)},
			{ID: "lampSwitcher_2", Type: "lampSwitcher", Info: json.RawMessage(`{"id":"lampSwitcher_2","delay":0.0}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
			{ID: "lamp_2", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_2","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "lampSwitcher_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
			{EntityID: "lampSwitcher_2", Edges: []api.EdgeDTO{{ToID: "lamp_2"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "lampSwitcher_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "lampSwitcher_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventInDTO{inputEvent(t, "lampSwitcher_2", true)}))
	steps = append(steps, tick(t, conn, reqID, 4, nil))
	steps = append(steps, tick(t, conn, reqID, 5, nil))
	steps = append(steps, tick(t, conn, reqID, 6, nil))
	steps = append(steps, tick(t, conn, reqID, 7, nil))

	lamp1State, _ := lastStateOf(steps, "lamp_1")
	lamp2State, _ := lastStateOf(steps, "lamp_2")

	if lamp1State != false {
		t.Fatalf("lamp_1 expected OFF, got ON")
	}
	if lamp2State != true {
		t.Fatalf("lamp_2 expected ON, got OFF")
	}
}

// ===== Tests for LightSwitchOffSensor =====
// Обычный сценарий: сенсор получил сигнал, потом завершил таймаут и выключился.
func TestSimulation_LightSwitchOffSensor_NoInterruption(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-sensor-no-interrupt"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Apartment: api.FieldDTO{Width: 2, Height: 2, Cells: [][]*api.CellDTO{}},
		Devices: []api.EntityDTO{
			{ID: "lightSwitchOffSensor_1", Type: "lightSwitchOffSensor", Info: json.RawMessage(`{"id":"lightSwitchOffSensor_1","delay":0.5,"timeout":1.0,"turned_on":false}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.5,"turned_on":false}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "lightSwitchOffSensor_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "lightSwitchOffSensor_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "lightSwitchOffSensor_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, nil))
	steps = append(steps, tick(t, conn, reqID, 4, nil))

	got := statesFrom(steps, "lightSwitchOffSensor_1")
	want := []bool{true, false}

	if len(got) < len(want) {
		t.Fatalf("not enough state changes for sensor: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sensor state mismatch at index %d: got=%v want=%v (full: %v)", i, got[i], want[i], got)
		}
	}
}

// Сценарий с 2 прерываниями: каждое новое срабатывание продлевает время работы сенсора.
func TestSimulation_LightSwitchOffSensor_TwoInterruptions(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-sensor-two-interrupts"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Apartment: api.FieldDTO{Width: 2, Height: 2, Cells: [][]*api.CellDTO{}},
		Devices: []api.EntityDTO{
			{ID: "lightSwitchOffSensor_1", Type: "lightSwitchOffSensor", Info: json.RawMessage(`{"id":"lightSwitchOffSensor_1","delay":0.0,"timeout":4.0,"turned_on":false}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0,"turned_on":false}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "lightSwitchOffSensor_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "lightSwitchOffSensor_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "lightSwitchOffSensor_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventInDTO{
		inputEvent(t, "lightSwitchOffSensor_1", true),
		inputEvent(t, "lightSwitchOffSensor_1", false),
	}))
	steps = append(steps, tick(t, conn, reqID, 4, nil))
	steps = append(steps, tick(t, conn, reqID, 5, []api.EventInDTO{
		inputEvent(t, "lightSwitchOffSensor_1", true),
		inputEvent(t, "lightSwitchOffSensor_1", false),
	}))
	steps = append(steps, tick(t, conn, reqID, 6, nil))
	steps = append(steps, tick(t, conn, reqID, 7, nil))
	steps = append(steps, tick(t, conn, reqID, 8, nil))
	steps = append(steps, tick(t, conn, reqID, 9, nil))
	steps = append(steps, tick(t, conn, reqID, 10, nil))

	got := statesFrom(steps, "lightSwitchOffSensor_1")
	want := []bool{true, false}

	if len(got) < len(want) {
		t.Fatalf("not enough state changes for sensor: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sensor state mismatch at index %d: got=%v want=%v (full: %v)", i, got[i], want[i], got)
		}
	}
}
