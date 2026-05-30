package simulations

import (
	"encoding/json"
	"fmt"
	"log"
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

	deviceName := strings.Split(entityID, "_")[0]

	payload, err := json.Marshal(map[string]any{"kind": fmt.Sprintf("%s:state", deviceName), "turn_on": turnOn})
	if err != nil {
		t.Fatalf("inputEvent: %v", err)
	}

	return api.EventInDTO{
		EntityID: entityID,
		Payload:  payload,
	}
}

func mockApartmentRaw(t *testing.T) json.RawMessage {
	t.Helper()
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
	raw, err := json.Marshal(floorObj)
	if err != nil {
		t.Fatalf("mockApartmentRaw: %v", err)
	}
	return raw
}

// statesFrom собирает последовательность turn_on из StateChanges для конкретного entityID.
func statesFrom(steps []api.SimulationStepPayload, entityID string) []bool {
	var result []bool
	for _, step := range steps {
		for _, change := range step.StateChanges {

			if change.EntityID != entityID {
				continue
			}

			var out struct {
				Kind   string `json:"kind"`
				TurnOn bool   `json:"turn_on"`
			}
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
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "switcher_1", Type: "switcher", Info: json.RawMessage(`{"id":"switcher_1", "delay":1.0}`)},
			{ID: "switcher_2", Type: "switcher", Info: json.RawMessage(`{"id":"switcher_2", "delay":0.3}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1", "delay":0.5}`)},
			{ID: "lamp_2", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_2", "delay":1.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "switcher_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
			{EntityID: "switcher_2", Edges: []api.EdgeDTO{{ToID: "lamp_1"}, {ToID: "lamp_2"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "switcher_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "switcher_2", true)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventInDTO{inputEvent(t, "switcher_1", false)}))
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
		DtSim:     10.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "switcher_1", Type: "switcher", Info: json.RawMessage(`{"id":"switcher_1","delay":0.0}`)},
			{ID: "switcher_2", Type: "switcher", Info: json.RawMessage(`{"id":"switcher_2","delay":0.0}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
			{ID: "lamp_2", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_2","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "switcher_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
			{EntityID: "switcher_2", Edges: []api.EdgeDTO{{ToID: "lamp_2"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "switcher_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "switcher_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventInDTO{inputEvent(t, "switcher_2", true)}))
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

// ===== Tests for SensorWithUpdate =====
// Обычный сценарий: сенсор получил сигнал, потом завершил таймаут и выключился.
func TestSimulation_SensorWithUpdate_NoInterruption(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-sensor-no-interrupt"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithUpdate_1", Type: "sensorWithUpdate", Info: json.RawMessage(`{"id":"sensorWithUpdate_1","delay":0.5,"timeout":1.0,"turn_on":false}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.5,"turn_on":false}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithUpdate_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "sensorWithUpdate_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "sensorWithUpdate_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, nil))
	steps = append(steps, tick(t, conn, reqID, 4, nil))

	got := statesFrom(steps, "sensorWithUpdate_1")
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
func TestSimulation_SensorWithUpdate_TwoInterruptions(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-sensor-two-interrupts"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithUpdate_1", Type: "sensorWithUpdate", Info: json.RawMessage(`{"id":"sensorWithUpdate_1","delay":0.0,"timeout":4.0,"turn_on":false}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0,"turn_on":false}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithUpdate_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{inputEvent(t, "sensorWithUpdate_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{inputEvent(t, "sensorWithUpdate_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventInDTO{
		inputEvent(t, "sensorWithUpdate_1", true),
		inputEvent(t, "sensorWithUpdate_1", false),
	}))
	steps = append(steps, tick(t, conn, reqID, 4, nil))
	steps = append(steps, tick(t, conn, reqID, 5, []api.EventInDTO{
		inputEvent(t, "sensorWithUpdate_1", true),
		inputEvent(t, "sensorWithUpdate_1", false),
	}))
	steps = append(steps, tick(t, conn, reqID, 6, nil))
	steps = append(steps, tick(t, conn, reqID, 7, nil))
	steps = append(steps, tick(t, conn, reqID, 8, nil))
	steps = append(steps, tick(t, conn, reqID, 9, nil))
	steps = append(steps, tick(t, conn, reqID, 10, nil))

	got := statesFrom(steps, "sensorWithUpdate_1")
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

// ===== Device interaction helpers =====

func boolEvent(t *testing.T, entityID string, field string, value bool) api.EventInDTO {
	t.Helper()
	payload, err := json.Marshal(map[string]bool{field: value})
	if err != nil {
		t.Fatalf("boolEvent: %v", err)
	}
	return api.EventInDTO{EntityID: entityID, Payload: payload}
}

func intEvent(t *testing.T, entityID string, field string, value int) api.EventInDTO {
	t.Helper()
	payload, err := json.Marshal(map[string]int{field: value})
	if err != nil {
		t.Fatalf("intEvent: %v", err)
	}
	return api.EventInDTO{EntityID: entityID, Payload: payload}
}

func lastBoolStateOf(steps []api.SimulationStepPayload, entityID string, field string) (bool, bool) {
	for i := len(steps) - 1; i >= 0; i-- {
		for _, change := range steps[i].StateChanges {
			if change.EntityID != entityID {
				continue
			}
			var out map[string]any
			if err := json.Unmarshal(change.Payload, &out); err != nil {
				continue
			}
			if v, ok := out[field]; ok {
				if b, ok := v.(bool); ok {
					return b, true
				}
			}
		}
	}
	return false, false
}

func lastIntStateOf(steps []api.SimulationStepPayload, entityID string, field string) (int, bool) {
	for i := len(steps) - 1; i >= 0; i-- {
		for _, change := range steps[i].StateChanges {
			if change.EntityID != entityID {
				continue
			}
			var out map[string]any
			if err := json.Unmarshal(change.Payload, &out); err != nil {
				continue
			}
			if v, ok := out[field]; ok {
				if f, ok := v.(float64); ok {
					return int(f), true
				}
			}
		}
	}
	return 0, false
}

// ===== Individual device tests =====

// TestDevice_MotionSensor_TriggersBulbAndSiren проверяет что датчик движения тригерит лампу и сирену.
func TestDevice_MotionSensor_TriggersBulbAndSiren(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-motion"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithUpdate_1", Type: "motion_sensor", Info: json.RawMessage(`{"id":"sensorWithUpdate_1","delay":0.0}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
			{ID: "siren_1", Type: "smart_siren", Info: json.RawMessage(`{"id":"siren_1","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithUpdate_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}, {ToID: "siren_1"}}},
		},
	})

	var steps []api.SimulationStepPayload
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{boolEvent(t, "sensorWithUpdate_1", "turn_on", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	bulbState, bulbFound := lastBoolStateOf(steps, "lamp_1", "turn_on")
	sirenState, sirenFound := lastBoolStateOf(steps, "siren_1", "turn_on")

	if !bulbFound || !bulbState {
		t.Fatal("lamp_1 should be ON after motion sensor trigger")
	}
	if !sirenFound || !sirenState {
		t.Fatal("siren_1 should be active after motion sensor trigger")
	}
}

// TestDevice_SmartDimmer_TriggersSmartLamp проверяет что умный диммер выставляет свет умной лампе.
func TestDevice_SmartDimmer_TriggersSmartLamp(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-presence"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "smartLamp_1", Type: "smart_bulb", Info: json.RawMessage(`{"id":"smartLamp_1","delay":0.0}`)},
			{ID: "smartDimmer_1", Type: "smart_dimmer", Info: json.RawMessage(`{"id":"smartDimmer_1","delay":0.0,"percents":0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "smartDimmer_1", Edges: []api.EdgeDTO{{ToID: "smartLamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{intEvent(t, "smartDimmer_1", "percents", 60)}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	smartLampState, smartLampFound := lastIntStateOf(steps, "smartLamp_1", "percents")
	if smartLampState != 60 || !smartLampFound {
		t.Fatal("smartLamp_1 should be 60 percents after smartDimmer_1 trigger")
	}
}

// TestDevice_SensorWithIntStatus_TriggersCurtains проверяет умные шторы.
func TestDevice_SensorWithIntStatus_TriggersCurtains(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-illumination"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithIntStatus_1", Type: "illumination_sensor", Info: json.RawMessage(`{"id":"sensorWithoutUpdate_1","delay":0.0}`)},
			{ID: "smartCurtains_1", Type: "curtains", Info: json.RawMessage(`{"id":"smartCurtains_1","delay":0.0,"percents":0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithIntStatus_1", Edges: []api.EdgeDTO{{ToID: "smartCurtains_1"}}},
		},
	})

	var steps []api.SimulationStepPayload
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{intEvent(t, "sensorWithIntStatus_1", "percents", 80)}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	smartCurtainsState, smartCurtainsFound := lastIntStateOf(steps, "smartCurtains_1", "percents")
	if smartCurtainsState != 80 || !smartCurtainsFound {
		t.Fatal("smartCurtains_1 should be 80% after sensorWithIntStatus_1 trigger")
	}
}

// TestDevice_DoorSensor_TriggersBulbLockSiren проверяет датчик двери.
func TestDevice_DoorSensor_TriggersBulbLockSiren(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-door-sensor"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithoutUpdate_1", Type: "door_sensor", Info: json.RawMessage(`{"id":"sensorWithoutUpdate_1","delay":0.0}`)},
			{ID: "lamp_1", Type: "smart_bulb", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
			{ID: "smartLock_1", Type: "smart_lock", Info: json.RawMessage(`{"id":"smartLock_1","delay":0.0}`)},
			{ID: "siren_1", Type: "smart_siren", Info: json.RawMessage(`{"id":"siren_1","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithoutUpdate_1", Edges: []api.EdgeDTO{
				{ToID: "lamp_1"},
				{ToID: "smartLock_1"},
				{ToID: "siren_1"},
			}},
		},
	})

	var steps []api.SimulationStepPayload
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{boolEvent(t, "sensorWithoutUpdate_1", "turn_on", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	state, found := lastBoolStateOf(steps, "lamp_1", "turn_on")
	if !found || !state {
		t.Fatal("lamp_1 should be ON after door sensor trigger")
	}
	state, found = lastBoolStateOf(steps, "smartLock_1", "turn_on")
	if !found || !state {
		t.Fatal("smartLock_1 should be ON after door sensor trigger")
	}
	state, found = lastBoolStateOf(steps, "siren_1", "turn_on")
	if !found || !state {
		t.Fatal("siren_1 should be ON after door sensor trigger")
	}
}

// TestDevice_WindowSensor_TriggersWindow проверяет датчик окна.
func TestDevice_WindowSensor_TriggersWindow(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-window-sensor"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithoutUpdate_1", Type: "window_sensor", Info: json.RawMessage(`{"id":"sensorWithoutUpdate_1","delay":0.0}`)},
			{ID: "window_1", Type: "window", Info: json.RawMessage(`{"id":"window_1","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithoutUpdate_1", Edges: []api.EdgeDTO{{ToID: "window_1"}}},
		},
	})

	var steps []api.SimulationStepPayload
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{boolEvent(t, "sensorWithoutUpdate_1", "turn_on", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	state, found := lastBoolStateOf(steps, "window_1", "turn_on")
	if !found || !state {
		t.Fatal("window_1 should be active after window sensor trigger")
	}
}

// TestDevice_Doorbell_TriggersLock проверяет умный дверной звонок.
func TestDevice_Doorbell_TriggersLock(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-doorbell"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "smartDoorbell_1", Type: "smart_doorbell", Info: json.RawMessage(`{"id":"smartDoorbell_1","delay":0.0}`)},
			{ID: "smartLock_1", Type: "smart_lock", Info: json.RawMessage(`{"id":"smartLock_1","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "smartDoorbell_1", Edges: []api.EdgeDTO{{ToID: "smartLock_1"}}},
		},
	})

	var steps []api.SimulationStepPayload
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{boolEvent(t, "smartDoorbell_1", "turn_on", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	lockState, lockFound := lastBoolStateOf(steps, "smartLock_1", "turn_on")
	if !lockFound || !lockState {
		t.Fatal("lock_1 should have received state change after doorbell trigger")
	}
}

// ===== Chain trigger test =====

// TestDevice_ChainTrigger проверяет длинную цепочку тригеров:
// motion_sensor -> smart_lock -> smart_lamp
func TestDevice_ChainTrigger(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-chain"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithoutUpdate_1", Type: "door_sensor", Info: json.RawMessage(`{"id":"sensorWithoutUpdate_1","delay":0.0}`)},
			{ID: "smartLock_1", Type: "smart_lock", Info: json.RawMessage(`{"id":"smartLock_1","delay":0.0}`)},
			{ID: "lamp_1", Type: "smart_bulb", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			// door_sensor → lock, camera
			{EntityID: "sensorWithoutUpdate_1", Edges: []api.EdgeDTO{{ToID: "smartLock_1"}}},
			// lock → bulb
			{EntityID: "smartLock_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{boolEvent(t, "sensorWithoutUpdate_1", "turn_on", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))
	steps = append(steps, tick(t, conn, reqID, 4, nil))

	// уровень 1: door_sensor тригерит lock и camera
	state, found := lastBoolStateOf(steps, "smartLock_1", "turn_on")
	if !found || !state {
		t.Fatal("smartLock_1 should have received state change (chain level 1)")
	}

	// уровень 2: lock тригерит lamp
	state, found = lastBoolStateOf(steps, "lamp_1", "turn_on")
	if !found || !state {
		t.Fatal("lamp_1 should have received state change (chain level 2 via lock)")
	}
}

// ===== Floor helpers =====

// mockFloorTwoRooms создаёт план с двумя комнатами соединёнными дверью.
func mockFloorTwoRooms(t *testing.T) json.RawMessage {
	t.Helper()

	floor := api.Floor{
		Meta: struct {
			Units string `json:"units"`
		}{Units: "meters"},
		Walls: []api.Wall{
			// стены room_1
			{ID: "w_r1_bottom", Points: [2][2]float64{{0, 0}, {5, 0}}, Width: 0.1},
			{ID: "w_shared", Points: [2][2]float64{{5, 0}, {5, 5}}, Width: 0.1}, // общая стена с дверью
			{ID: "w_r1_top", Points: [2][2]float64{{5, 5}, {0, 5}}, Width: 0.1},
			{ID: "w_r1_left", Points: [2][2]float64{{0, 5}, {0, 0}}, Width: 0.1},
			// стены room_2
			{ID: "w_r2_bottom", Points: [2][2]float64{{5, 0}, {10, 0}}, Width: 0.1},
			{ID: "w_r2_right", Points: [2][2]float64{{10, 0}, {10, 5}}, Width: 0.1},
			{ID: "w_r2_top", Points: [2][2]float64{{10, 5}, {5, 5}}, Width: 0.1},
		},
		Doors: []api.Door{
			{
				ID:     "door_1",
				Points: [2][2]float64{{5, 2}, {5, 3}},
				Width:  1.0,
				Rooms:  []string{"room_1", "room_2"},
			},
		},
		Rooms: []api.Room{
			{
				ID:    "room_1",
				Name:  "Living Room",
				Area:  [][2]float64{{0, 0}, {5, 0}, {5, 5}, {0, 5}},
				Walls: []string{"w_r1_bottom", "w_shared", "w_r1_top", "w_r1_left"},
				Doors: []string{"door_1"},
			},
			{
				ID:    "room_2",
				Name:  "Bedroom",
				Area:  [][2]float64{{5, 0}, {10, 0}, {10, 5}, {5, 5}},
				Walls: []string{"w_shared", "w_r2_bottom", "w_r2_right", "w_r2_top"},
				Doors: []string{"door_1"},
			},
		},
	}

	raw, err := json.Marshal(floor)
	if err != nil {
		t.Fatalf("mockFloorTwoRooms: %v", err)
	}
	return raw
}

// ===== Human input helpers =====

func humanMoveInput(t *testing.T, humanID string, x, y float64) api.EventInDTO {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"kind": "human:move",
		"to":   map[string]float64{"x": x, "y": y},
	})
	if err != nil {
		t.Fatalf("humanMoveInput: %v", err)
	}

	return api.EventInDTO{
		EntityID: humanID,
		Payload:  payload,
	}
}

func humanInteractionInput(t *testing.T, humanID string, deviceID string, devicePayload any) api.EventInDTO {
	t.Helper()

	rawDevice, err := json.Marshal(devicePayload)
	if err != nil {
		t.Fatalf("humanInteractionInput marshal device payload: %v", err)
	}

	payload, err := json.Marshal(struct {
		Kind          string          `json:"kind"`
		DeviceID      string          `json:"device_id"`
		DevicePayload json.RawMessage `json:"device_payload"`
	}{
		Kind:          "human:interaction",
		DeviceID:      deviceID,
		DevicePayload: rawDevice,
	})
	if err != nil {
		t.Fatalf("humanInteractionInput: %v", err)
	}

	return api.EventInDTO{
		EntityID: humanID,
		Payload:  payload,
	}
}

// humanPositionFrom возвращает последнюю позицию человека из StateChanges.
func humanPositionFrom(steps []api.SimulationStepPayload, humanID string) (x, y float64, found bool) {
	for i := len(steps) - 1; i >= 0; i-- {
		for _, change := range steps[i].StateChanges {
			if change.EntityID != humanID {
				continue
			}
			var out struct {
				To struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"to"`
				Status string `json:"status"`
			}
			if err := json.Unmarshal(change.Payload, &out); err != nil {
				continue
			}
			if out.Status == "moved" {
				return out.To.X, out.To.Y, true
			}
		}
	}
	return 0, 0, false
}

// humanRoomFrom возвращает последнюю комнату человека из StateChanges.
func humanRoomFrom(steps []api.SimulationStepPayload, humanID string) (roomID string, found bool) {
	for i := len(steps) - 1; i >= 0; i-- {
		for _, change := range steps[i].StateChanges {
			if change.EntityID != humanID {
				continue
			}
			var out struct {
				To struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"to"`
				RoomID string `json:"roomID"`
				Status string `json:"status"`
			}
			if err := json.Unmarshal(change.Payload, &out); err != nil {
				continue
			}
			log.Println("Changes: ", out.To.X, out.To.Y, out.RoomID, out.Status)
			if out.RoomID != "" {
				return out.RoomID, true
			}
		}
	}
	return "", false
}

// ===== Tests for human =====

// TestHuman_NormalMove проверяет нормальное движение внутри комнаты.
func TestHuman_NormalMove(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-move"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRooms(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":1.0,"y":1.0,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	// двигаем человека в центр room_1
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{
		humanMoveInput(t, "human_1", 2.5, 2.5),
	}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))

	x, y, found := humanPositionFrom(steps, "human_1")
	if !found {
		t.Fatal("no position found for human_1")
	}

	// человек должен дойти до целевой точки
	if x != 2.5 || y != 2.5 {
		t.Fatalf("expected position (2.5, 2.5), got (%.2f, %.2f)", x, y)
	}
}

// TestHuman_BlockedByWall проверяет что человек не проходит через стену.
func TestHuman_BlockedByWall(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-wall"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRooms(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":2.5,"y":4.0,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	// пытаемся пройти через стену x=5
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{
		humanMoveInput(t, "human_1", 7.5, 4.0),
	}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))

	x, _, found := humanPositionFrom(steps, "human_1")
	if !found {
		t.Fatal("no position found for human_1")
	}

	// человек должен остановиться перед стеной x=5, не пройдя в room_2
	if x >= 5.0 {
		t.Fatalf("human should be blocked by wall at x=5, got x=%.2f", x)
	}

	// комната не должна измениться
	roomID, roomFound := humanRoomFrom(steps, "human_1")
	if roomFound && roomID != "room_1" {
		t.Fatalf("human should stay in room_1, got %s", roomID)
	}
}

// TestHuman_MoveThroughDoor проверяет что человек проходит через дверь и меняет комнату.
func TestHuman_MoveThroughDoor(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-door"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRooms(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":2.5,"y":2.5,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	// двигаемся через дверь (5,2)-(5,3) в room_2
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{
		humanMoveInput(t, "human_1", 7.5, 2.5),
	}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))

	x, _, found := humanPositionFrom(steps, "human_1")
	if !found {
		t.Fatal("no position found for human_1")
	}

	// человек должен оказаться в room_2 (x > 5)
	if x <= 5.0 {
		t.Fatalf("human should have passed through door into room_2, got x=%.2f", x)
	}

	roomID, roomFound := humanRoomFrom(steps, "human_1")
	if !roomFound {
		t.Fatal("no room found for human_1")
	}
	if roomID != "room_2" {
		t.Fatalf("expected room_2, got %s", roomID)
	}
}

func TestHuman_InteractionWithLamp(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-lamp"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRooms(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":2.5,"y":2.5,"roomID":"room_1"}`),
			},
			{
				ID:   "lamp_1",
				Type: "lamp",
				Info: json.RawMessage(`{"id":"lamp_1","delay":0.0,"turn_on":false}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{
		humanInteractionInput(t, "human_1", "lamp_1", map[string]any{
			"kind":    "lamp:state",
			"turn_on": true,
		}),
	}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	lampState, found := lastStateOf(steps, "lamp_1")
	if !found {
		t.Fatal("no state change found for lamp_1")
	}
	if !lampState {
		t.Fatal("lamp_1 should be ON after human interaction")
	}
}

func TestHuman_InteractionThenMove(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-interact-move"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRooms(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":1.0,"y":1.0,"roomID":"room_1"}`),
			},
			{
				ID:   "lamp_1",
				Type: "lamp",
				Info: json.RawMessage(`{"id":"lamp_1","delay":0.0,"turn_on":false}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{
		humanInteractionInput(t, "human_1", "lamp_1", map[string]any{
			"kind":    "lamp:state",
			"turn_on": true,
		}),
	}))

	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{
		humanMoveInput(t, "human_1", 3.0, 3.0),
	}))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	lampState, found := lastStateOf(steps, "lamp_1")
	if !found {
		t.Fatal("no state change for lamp_1")
	}
	if !lampState {
		t.Fatal("lamp_1 should be ON")
	}

	x, y, posFound := humanPositionFrom(steps, "human_1")
	if !posFound {
		t.Fatal("no position found for human_1")
	}
	if x != 3.0 || y != 3.0 {
		t.Fatalf("expected position (3.0, 3.0), got (%.2f, %.2f)", x, y)
	}
}
