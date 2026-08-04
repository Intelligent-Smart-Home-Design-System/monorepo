package simulations

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/client/ws"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/gorilla/websocket"
)

// ===== Helper =====
// incidentTestBlock описывает минимальный DTO incident-блока, нужный e2e-тестам.
type incidentTestBlock struct {
	Points [][2]float64 `json:"points"`
}

// incidentBlockCount возвращает число блоков заданного incident в одном simulation:step.
func incidentBlockCount(step api.SimulationStepPayload, entityID, kind string) (int, bool) {
	for _, change := range step.StateChanges {
		if change.EntityID != entityID {
			continue
		}

		var out struct {
			Kind      string `json:"kind"`
			Incidents []struct {
				Blocks []incidentTestBlock `json:"blocks"`
			} `json:"incidents"`
		}
		if err := json.Unmarshal(change.Payload, &out); err != nil || out.Kind != kind {
			continue
		}

		count := 0
		for _, zone := range out.Incidents {
			count += len(zone.Blocks)
		}
		return count, true
	}

	return 0, false
}

// dialSim устанавливает WebSocket-соединение
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

// incidentBlocksCoverPoint проверяет, покрывает ли хотя бы один incident-блок заданную точку.
func incidentBlocksCoverPoint(blocks []incidentTestBlock, point [2]float64) bool {
	for _, block := range blocks {
		if pointInPolygon(point, block.Points) {
			return true
		}
	}

	return false
}

// pointInPolygon проверяет, находится ли точка внутри polygon или на его границе.
func pointInPolygon(point [2]float64, polygon [][2]float64) bool {
	if len(polygon) < 3 {
		return false
	}

	for i := range polygon {
		next := (i + 1) % len(polygon)
		if pointOnSegment(point, polygon[i], polygon[next]) {
			return true
		}
	}

	inside := false
	j := len(polygon) - 1
	for i := range polygon {
		pi := polygon[i]
		pj := polygon[j]
		if (pi[1] > point[1]) != (pj[1] > point[1]) {
			x := (pj[0]-pi[0])*(point[1]-pi[1])/(pj[1]-pi[1]) + pi[0]
			if point[0] < x {
				inside = !inside
			}
		}
		j = i
	}

	return inside
}

// pointOnSegment проверяет, лежит ли точка на отрезке ab.
func pointOnSegment(point, a, b [2]float64) bool {
	const epsilon = 1e-9
	cross := (point[1]-a[1])*(b[0]-a[0]) - (point[0]-a[0])*(b[1]-a[1])
	if cross < -epsilon || cross > epsilon {
		return false
	}

	dot := (point[0]-a[0])*(b[0]-a[0]) + (point[1]-a[1])*(b[1]-a[1])
	if dot < -epsilon {
		return false
	}

	lengthSquared := (b[0]-a[0])*(b[0]-a[0]) + (b[1]-a[1])*(b[1]-a[1])
	return dot <= lengthSquared+epsilon
}

// sendMsg отправляет сообщение msg через websocket соединение conn. Если возникает ошибка, тест завершается с фатальной ошибкой.
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

// recvMsg читает сообщение из websocket соединения conn, десериализует его в api.Message и возвращает. Если возникает ошибка при чтении или десериализации, тест завершается с фатальной ошибкой.
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

// recvStep читает сообщение из websocket соединения conn, проверяет что его тип "simulation:step",
// десериализует полезную нагрузку в api.SimulationStepPayload и возвращает её. Если тип сообщения
// не соответствует ожидаемому или возникает ошибка при чтении или десериализации, тест завершается с фатальной ошибкой.
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

// newSimServer создает новый тестовый HTTP сервер, который обрабатывает WebSocket соединения с помощью ws.Manager.
func newSimServer(t *testing.T) *httptest.Server {
	t.Helper()

	simService := NewSimulation()
	manager := ws.NewManager(simService)
	server := httptest.NewServer(http.HandlerFunc(manager.ServeWS))
	t.Cleanup(server.Close)

	return server
}

// startSim отправляет команду "simulation:start" с заданным reqID и полезной нагрузкой payload через
// websocket соединение conn.
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

// TestHeartbeat проверяет, что служебный ping получает pong без запуска или тика симуляции.
func TestHeartbeat(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	reqID := "heartbeat-test"

	sendMsg(t, conn, api.Message{Type: "ping", Ts: time.Now(), ReqID: reqID})
	msg := recvMsg(t, conn)

	if msg.Type != "pong" {
		t.Fatalf("expected pong, got %q", msg.Type)
	}
	if msg.ReqID != reqID {
		t.Fatalf("expected reqID %q, got %q", reqID, msg.ReqID)
	}
}

// TestHelloAckPreservesReqID проверяет, что hello:ack относится к той же клиентской сессии.
func TestHelloAckPreservesReqID(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	reqID := "hello-contract-test"
	payload, _ := json.Marshal(api.HelloPayload{Client: "sim-ui", Version: "0.1.0"})

	sendMsg(t, conn, api.Message{Type: "hello", Ts: time.Now(), ReqID: reqID, Payload: payload})
	msg := recvMsg(t, conn)

	if msg.Type != "hello:ack" {
		t.Fatalf("expected hello:ack, got %q", msg.Type)
	}
	if msg.ReqID != reqID {
		t.Fatalf("expected reqID %q, got %q", reqID, msg.ReqID)
	}
}

// TestWebSocketDisconnectRemovesBackendSession проверяет немедленное удаление
// движка после закрытия владеющего им WebSocket.
func TestWebSocketDisconnectRemovesBackendSession(t *testing.T) {
	simService := NewSimulation()
	manager := ws.NewManager(simService)
	server := httptest.NewServer(http.HandlerFunc(manager.ServeWS))
	t.Cleanup(server.Close)
	conn := dialSim(t, server)
	const reqID = "disconnect-cleanup-test"
	startSim(t, conn, reqID, validStartPayload())
	if err := conn.Close(); err != nil {
		t.Fatalf("close websocket: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		simService.mu.RLock()
		_, exists := simService.sessions[reqID]
		simService.mu.RUnlock()
		if !exists {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("backend session still exists after websocket disconnect")
}

// TestInvalidJSONDoesNotCloseConnection проверяет, что ошибочное сообщение не разрывает WebSocket.
func TestInvalidJSONDoesNotCloseConnection(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":`)); err != nil {
		t.Fatalf("write invalid message: %v", err)
	}
	msg := recvMsg(t, conn)
	if msg.Type != "error" {
		t.Fatalf("expected error, got %q", msg.Type)
	}

	sendMsg(t, conn, api.Message{Type: "ping", Ts: time.Now(), ReqID: "after-invalid-json"})
	msg = recvMsg(t, conn)
	if msg.Type != "pong" {
		t.Fatalf("expected pong after invalid JSON, got %q", msg.Type)
	}
}

// TestSimulationCommandsValidateContractFields проверяет обязательный reqId и положительные dtSim/tick.
func TestSimulationCommandsValidateContractFields(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	startPayload := validStartPayload()

	raw, _ := json.Marshal(startPayload)
	sendMsg(t, conn, api.Message{Type: "simulation:start", Ts: time.Now(), Payload: raw})
	assertErrorCode(t, recvMsg(t, conn), "MISSING_REQ_ID")

	startPayload.DtSim = 0
	raw, _ = json.Marshal(startPayload)
	sendMsg(t, conn, api.Message{Type: "simulation:start", Ts: time.Now(), ReqID: "invalid-dt", Payload: raw})
	assertErrorCode(t, recvMsg(t, conn), "INVALID_PAYLOAD")

	startSim(t, conn, "invalid-tick", validStartPayload())
	raw, _ = json.Marshal(api.SimulationTickPayload{Tick: 0})
	sendMsg(t, conn, api.Message{Type: "simulation:tick", Ts: time.Now(), ReqID: "invalid-tick", Payload: raw})
	assertErrorCode(t, recvMsg(t, conn), "INVALID_PAYLOAD")
}

// TestInvalidTickInputStopsSession проверяет явную ошибку неизвестной сущности и остановку неконсистентной сессии.
func TestInvalidTickInputStopsSession(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "invalid-tick-input"

	startSim(t, conn, reqID, validStartPayload())

	raw, err := json.Marshal(api.SimulationTickPayload{
		Tick:   1,
		Inputs: []api.EventDTO{{EntityID: "missing", Payload: json.RawMessage(`{"turn_on":true}`)}},
	})
	if err != nil {
		t.Fatalf("marshal invalid tick: %v", err)
	}
	sendMsg(t, conn, api.Message{Type: "simulation:tick", Ts: time.Now(), ReqID: reqID, Payload: raw})
	assertErrorCode(t, recvMsg(t, conn), "TICK_FAILED")

	raw, _ = json.Marshal(api.SimulationTickPayload{Tick: 2})
	sendMsg(t, conn, api.Message{Type: "simulation:tick", Ts: time.Now(), ReqID: reqID, Payload: raw})
	assertErrorCode(t, recvMsg(t, conn), "SESSION_NOT_ATTACHED")
}

// TestSimulationStepUsesEmptyCollections проверяет, что массивы контракта кодируются как [], а не null.
func TestSimulationStepUsesEmptyCollections(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "empty-step-collections"

	startSim(t, conn, reqID, validStartPayload())
	step := tick(t, conn, reqID, 1, nil)

	if step.StateChanges == nil {
		t.Fatal("stateChanges must be an empty array, got nil")
	}
	if step.TriggeredEdges == nil {
		t.Fatal("triggeredEdges must be an empty array, got nil")
	}
	if step.Humans == nil {
		t.Fatal("humans must be an empty array, got nil")
	}
}

// assertErrorCode проверяет тип сообщения error и его машинный код.
func assertErrorCode(t *testing.T, msg api.Message, expected string) {
	t.Helper()
	if msg.Type != "error" {
		t.Fatalf("expected error, got %q", msg.Type)
	}

	var payload api.ErrorPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("unmarshal error payload: %v", err)
	}
	if payload.Code != expected {
		t.Fatalf("expected error code %q, got %q", expected, payload.Code)
	}
}

// tick отправляет команду "simulation:tick" с заданным reqID
func tick(t *testing.T, conn *websocket.Conn, reqID string, tickN int, inputs []api.EventDTO) api.SimulationStepPayload {
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

// inputEvent создает событие для включения или выключения устройства с данным entityID.
func inputEvent(t *testing.T, entityID string, turnOn bool) api.EventDTO {
	t.Helper()

	deviceName := strings.Split(entityID, "_")[0]

	payload, err := json.Marshal(map[string]any{"kind": fmt.Sprintf("%s:state", deviceName), "turn_on": turnOn})
	if err != nil {
		t.Fatalf("inputEvent: %v", err)
	}

	return api.EventDTO{
		EntityID: entityID,
		Payload:  payload,
	}
}

// mockApartmentRaw создает простое описание квартиры в виде json.RawMessage для использования в тестах.
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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{inputEvent(t, "switcher_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventDTO{inputEvent(t, "switcher_2", true)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventDTO{inputEvent(t, "switcher_1", false)}))
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

// TestSimulation_ReturnsTriggeredScenarioChain проверяет возврат всех постоянных связей, реально сработавших за tick.
func TestSimulation_ReturnsTriggeredScenarioChain(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-triggered-chain"
	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "switcher_1", Type: "switcher", Info: json.RawMessage(`{"id":"switcher_1","delay":0.0}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
			{ID: "lamp_2", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_2","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{
				EntityID: "switcher_1",
				Edges: []api.EdgeDTO{
					{ToID: "lamp_1", Action: "trigger", Data: []interface{}{"first"}},
				},
			},
			{
				EntityID: "lamp_1",
				Edges:    []api.EdgeDTO{{ToID: "lamp_2", Action: "trigger"}},
			},
		},
	})

	step := tick(t, conn, reqID, 1, []api.EventDTO{inputEvent(t, "switcher_1", true)})
	if len(step.TriggeredEdges) != 2 {
		t.Fatalf("expected two triggered edges, got %+v", step.TriggeredEdges)
	}

	first, second := step.TriggeredEdges[0], step.TriggeredEdges[1]
	if first.FromID != "switcher_1" || first.ToID != "lamp_1" || first.Action != "trigger" {
		t.Fatalf("unexpected first triggered edge: %+v", first)
	}
	if len(first.Data) != 1 || first.Data[0] != "first" {
		t.Fatalf("first edge data was not preserved: %+v", first.Data)
	}
	if second.FromID != "lamp_1" || second.ToID != "lamp_2" || second.Action != "trigger" {
		t.Fatalf("unexpected second triggered edge: %+v", second)
	}

	emptyStep := tick(t, conn, reqID, 2, nil)
	if len(emptyStep.TriggeredEdges) != 0 {
		t.Fatalf("triggered edges leaked into the next tick: %+v", emptyStep.TriggeredEdges)
	}
}

// Тест проверки корректности работы программы при вмешательстве пользователя.
func TestWS_Simulation_UserIntervention(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-intervention"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{inputEvent(t, "switcher_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventDTO{inputEvent(t, "switcher_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventDTO{inputEvent(t, "switcher_2", true)}))

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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{inputEvent(t, "sensorWithUpdate_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventDTO{inputEvent(t, "sensorWithUpdate_1", false)}))

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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{inputEvent(t, "sensorWithUpdate_1", true)}))
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventDTO{inputEvent(t, "sensorWithUpdate_1", false)}))
	steps = append(steps, tick(t, conn, reqID, 3, []api.EventDTO{
		inputEvent(t, "sensorWithUpdate_1", true),
		inputEvent(t, "sensorWithUpdate_1", false),
	}))
	steps = append(steps, tick(t, conn, reqID, 4, nil))
	steps = append(steps, tick(t, conn, reqID, 5, []api.EventDTO{
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

// boolEvent создает событие для устройства с данным entityID, устанавливая поле field в значение value.
func boolEvent(t *testing.T, entityID string, field string, value bool) api.EventDTO {
	t.Helper()

	payload, err := json.Marshal(map[string]bool{field: value})
	if err != nil {
		t.Fatalf("boolEvent: %v", err)
	}

	return api.EventDTO{EntityID: entityID, Payload: payload}
}

// intEvent создает событие для устройства с данным entityID, устанавливая поле field в значение value.
func intEvent(t *testing.T, entityID string, field string, value int) api.EventDTO {
	t.Helper()

	payload, err := json.Marshal(map[string]int{field: value})
	if err != nil {
		t.Fatalf("intEvent: %v", err)
	}

	return api.EventDTO{EntityID: entityID, Payload: payload}
}

// lastBoolStateOf возвращает последнее значение поля field для entityID из всех шагов. Если такого поля не найдено, возвращает false и false.
func lastBoolStateOf(steps []api.SimulationStepPayload, entityID string, field string) (bool, bool) {
	for i := len(steps) - 1; i >= 0; i-- {
		changes := steps[i].StateChanges
		for j := len(changes) - 1; j >= 0; j-- {
			change := changes[j]
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

// lastIntStateOf возвращает последнее значение поля field для entityID из всех шагов. Если такого поля не найдено, возвращает 0 и false.
func lastIntStateOf(steps []api.SimulationStepPayload, entityID string, field string) (int, bool) {
	for i := len(steps) - 1; i >= 0; i-- {
		changes := steps[i].StateChanges
		for j := len(changes) - 1; j >= 0; j-- {
			change := changes[j]
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

// TestDevice_MotionSensor_TriggersLampAndSiren проверяет что датчик движения тригерит лампу и сирену.
func TestDevice_MotionSensor_TriggersLampAndSiren(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-motion"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithUpdate_1", Type: "motion_sensor", Info: json.RawMessage(`{"id":"sensorWithUpdate_1","delay":0.0, "timeout": 1000}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
			{ID: "siren_1", Type: "smart_siren", Info: json.RawMessage(`{"id":"siren_1","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithUpdate_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}, {ToID: "siren_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{boolEvent(t, "sensorWithUpdate_1", "turn_on", true)}))

	lampState, lampFound := lastBoolStateOf(steps, "lamp_1", "turn_on")
	sirenState, sirenFound := lastBoolStateOf(steps, "siren_1", "turn_on")

	if !lampFound || !lampState {
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
			{ID: "smartLamp_1", Type: "smart_lamp", Info: json.RawMessage(`{"id":"smartLamp_1","delay":0.0}`)},
			{ID: "smartDimmer_1", Type: "smart_dimmer", Info: json.RawMessage(`{"id":"smartDimmer_1","delay":0.0,"percents":0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "smartDimmer_1", Edges: []api.EdgeDTO{{ToID: "smartLamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{intEvent(t, "smartDimmer_1", "percents", 60)}))

	smartLampState, smartLampFound := lastIntStateOf(steps, "smartLamp_1", "percents")
	if smartLampState != 60 || !smartLampFound {
		t.Fatal("smartLamp_1 should be 60 percents after smartDimmer_1 trigger")
	}
}

// TestDevice_SensorWithIntStatus_TriggersCurtains проверяет умные шторы через сенсор полем int.
func TestDevice_SensorWithIntStatus_TriggersCurtains(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-illumination"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithIntStatus_1", Type: "illumination_sensor", Info: json.RawMessage(`{"id":"sensorWithIntStatus_1","delay":0.0}`)},
			{ID: "smartCurtains_1", Type: "curtains", Info: json.RawMessage(`{"id":"smartCurtains_1","delay":0.0,"percents":0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithIntStatus_1", Edges: []api.EdgeDTO{{ToID: "smartCurtains_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{intEvent(t, "sensorWithIntStatus_1", "percents", 80)}))

	smartCurtainsState, smartCurtainsFound := lastIntStateOf(steps, "smartCurtains_1", "percents")
	if smartCurtainsState != 80 || !smartCurtainsFound {
		t.Fatal("smartCurtains_1 should be 80% after sensorWithIntStatus_1 trigger")
	}
}

// TestDevice_DoorSensor_TriggersLampLockSiren проверяет сенсор, лампу, дверной замок и сирену.
func TestDevice_DoorSensor_TriggersLampLockSiren(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-door-sensor"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
		Devices: []api.EntityDTO{
			{ID: "sensorWithoutUpdate_1", Type: "door_sensor", Info: json.RawMessage(`{"id":"sensorWithoutUpdate_1","delay":0.0}`)},
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{boolEvent(t, "sensorWithoutUpdate_1", "turn_on", true)}))

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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{boolEvent(t, "sensorWithoutUpdate_1", "turn_on", true)}))

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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{boolEvent(t, "smartDoorbell_1", "turn_on", true)}))

	lockState, lockFound := lastBoolStateOf(steps, "smartLock_1", "turn_on")
	if !lockFound || !lockState {
		t.Fatal("lock_1 should have received state change after doorbell trigger")
	}
}

// TestDevice_AirConditioner проверяет работу кондиционера.
func TestDevice_AirConditioner(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-ac"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Devices: []api.EntityDTO{
			{ID: "airConditioner_1", Type: "airConditioner", Info: json.RawMessage(`{"id":"airConditioner_1", "turn_on":false,"temperature":20, "delay":0.0}`)},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		boolEvent(t, "airConditioner_1", "turn_on", true),
	}))

	steps = append(steps, tick(t, conn, reqID, 2, []api.EventDTO{
		intEvent(t, "airConditioner_1", "temperature", 25),
	}))

	on, found := lastBoolStateOf(steps, "airConditioner_1", "turn_on")
	if !found || !on {
		t.Fatal("AirConditioner should be turned on")
	}

	temp, found := lastIntStateOf(steps, "airConditioner_1", "temperature")
	if !found || temp != 25 {
		t.Fatal("AirConditioner temperature should be 25")
	}
}

// TestDevice_Thermostat проверяет работу теромостата
func TestDevice_Thermostat(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-thermo"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Devices: []api.EntityDTO{
			{ID: "thermostat_1", Type: "thermostat", Info: json.RawMessage(`{"id":"thermostat_1", "turn_on":false,"temperature":0}`)},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		boolEvent(t, "thermostat_1", "turn_on", true),
		intEvent(t, "thermostat_1", "temperature", 75),
	}))

	on, found := lastBoolStateOf(steps, "thermostat_1", "turn_on")
	if !found || !on {
		t.Fatal("Thermostat should be turned on")
	}

	temperature, found := lastIntStateOf(steps, "thermostat_1", "temperature")
	if !found || temperature != 75 {
		t.Fatal("Thermostat temperature should be 75")
	}
}

// TestDevice_SmartFloor проверяет работу умного пола
func TestDevice_SmartFloor(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-floor"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Devices: []api.EntityDTO{
			{ID: "smartFloor_1", Type: entities.TypeSmartFloor, Info: json.RawMessage(`{"id":"smartFloor_1", "turn_on":false}`)},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		boolEvent(t, "smartFloor_1", "turn_on", true),
	}))

	on, found := lastBoolStateOf(steps, "smartFloor_1", "turn_on")
	if !found || !on {
		t.Fatal("SmartFloor should be turned on")
	}
}

// проверяет работу телевизора.
func TestDevice_TV(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-tv"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Devices: []api.EntityDTO{
			{ID: "tv_1", Type: entities.TypeTV, Info: json.RawMessage(`{"id":"tv_1", "turn_on":false}`)},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		boolEvent(t, "tv_1", "turn_on", true),
	}))

	on, found := lastBoolStateOf(steps, "tv_1", "turn_on")
	if !found || !on {
		t.Fatal("TV should be turned on")
	}
}

// TestDevice_Subwoofer проверяет работу сабвуфера.
func TestDevice_Subwoofer(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-sub"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim: 1.0,
		Devices: []api.EntityDTO{
			{ID: "subwoofer_1", Type: entities.TypeSubwoofer, Info: json.RawMessage(`{"id":"subwoofer_1", "turn_on":false}`)},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		boolEvent(t, "subwoofer_1", "turn_on", true),
	}))

	on, found := lastBoolStateOf(steps, "subwoofer_1", "turn_on")
	if !found || !on {
		t.Fatal("Subwoofer should be turned on")
	}
}

// TestDevice_ChainTrigger проверяет длинную цепочку тригеров:
// door_sensor -> smart_lock -> lamp
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
			{ID: "lamp_1", Type: "lamp", Info: json.RawMessage(`{"id":"lamp_1","delay":0.0}`)},
		},
		Scenarios: []api.ScenarioDTO{
			{EntityID: "sensorWithoutUpdate_1", Edges: []api.EdgeDTO{{ToID: "smartLock_1"}}},
			{EntityID: "smartLock_1", Edges: []api.EdgeDTO{{ToID: "lamp_1"}}},
		},
	})

	var steps []api.SimulationStepPayload

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{boolEvent(t, "sensorWithoutUpdate_1", "turn_on", true)}))

	state, found := lastBoolStateOf(steps, "smartLock_1", "turn_on")
	if !found || !state {
		t.Fatal("smartLock_1 should have received state change (chain level 1)")
	}

	state, found = lastBoolStateOf(steps, "lamp_1", "turn_on")
	if !found || !state {
		t.Fatal("lamp_1 should have received state change (chain level 2 via lock)")
	}
}

// ===== Floor helpers =====

// mockFloorOneRoom создаёт изолированную комнату без дверей.
func mockFloorOneRoom(t *testing.T) json.RawMessage {
	t.Helper()

	floor := api.Floor{
		Meta: struct {
			Units string `json:"units"`
		}{Units: "meters"},
		Walls: []api.Wall{
			{ID: "w_bottom", Points: [2][2]float64{{0, 0}, {4, 0}}, Width: 0.1},
			{ID: "w_right", Points: [2][2]float64{{4, 0}, {4, 4}}, Width: 0.1},
			{ID: "w_top", Points: [2][2]float64{{4, 4}, {0, 4}}, Width: 0.1},
			{ID: "w_left", Points: [2][2]float64{{0, 4}, {0, 0}}, Width: 0.1},
		},
		Rooms: []api.Room{
			{
				ID:    "room_1",
				Name:  "Isolated Room",
				Area:  [][2]float64{{0, 0}, {4, 0}, {4, 4}, {0, 4}},
				Walls: []string{"w_bottom", "w_right", "w_top", "w_left"},
			},
		},
	}

	raw, err := json.Marshal(floor)
	if err != nil {
		t.Fatalf("mockFloorOneRoom: %v", err)
	}
	return raw
}

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

// mockFloorTwoRoomsWithoutRoomWallRefs возвращает тот же план без room.walls, сохраняя геометрию room.area.
func mockFloorTwoRoomsWithoutRoomWallRefs(t *testing.T) json.RawMessage {
	t.Helper()

	var floor api.Floor
	if err := json.Unmarshal(mockFloorTwoRooms(t), &floor); err != nil {
		t.Fatalf("unmarshal floor without room wall refs: %v", err)
	}
	for index := range floor.Rooms {
		floor.Rooms[index].Walls = nil
	}

	raw, err := json.Marshal(floor)
	if err != nil {
		t.Fatalf("marshal floor without room wall refs: %v", err)
	}
	return raw
}

// mockFloorTwoRoomsWithOffsetDoor возвращает план, где линия двери смещена
// относительно room.area на толщину стены, как в планах floor-parser.
func mockFloorTwoRoomsWithOffsetDoor(t *testing.T) json.RawMessage {
	t.Helper()

	var floor api.Floor
	if err := json.Unmarshal(mockFloorTwoRooms(t), &floor); err != nil {
		t.Fatalf("unmarshal floor with offset door: %v", err)
	}
	floor.Doors[0].Points = [2][2]float64{{4.85, 2}, {4.85, 3}}

	raw, err := json.Marshal(floor)
	if err != nil {
		t.Fatalf("marshal floor with offset door: %v", err)
	}
	return raw
}

// mockFloorTwoRoomsWithDoorGap возвращает план, где room.area разделены
// полосой стены, а дверь расположена между ними.
func mockFloorTwoRoomsWithDoorGap(t *testing.T) json.RawMessage {
	t.Helper()

	var floor api.Floor
	if err := json.Unmarshal(mockFloorTwoRooms(t), &floor); err != nil {
		t.Fatalf("unmarshal floor with door gap: %v", err)
	}
	floor.Rooms[0].Area = [][2]float64{{0, 0}, {4.9, 0}, {4.9, 5}, {0, 5}}
	floor.Rooms[1].Area = [][2]float64{{5.1, 0}, {10, 0}, {10, 5}, {5.1, 5}}

	raw, err := json.Marshal(floor)
	if err != nil {
		t.Fatalf("marshal floor with door gap: %v", err)
	}
	return raw
}

// ===== Human input helpers =====

// humanMoveInput возвращает событие для пережвижение человека на основе входных humanID, x и y.
func humanMoveInput(t *testing.T, humanID string, x, y float64) api.EventDTO {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"kind": "human:move",
		"to":   map[string]float64{"x": x, "y": y},
	})
	if err != nil {
		t.Fatalf("humanMoveInput: %v", err)
	}

	return api.EventDTO{
		EntityID: humanID,
		Payload:  payload,
	}
}

// humanInteractionInput возвращает событие для взаимодействия человека на основе входных humanID, deviceID и devicePayload.
func humanInteractionInput(t *testing.T, humanID string, deviceID string, devicePayload any) api.EventDTO {
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

	return api.EventDTO{
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
				Kind string `json:"kind"`
				To   struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"to"`
			}
			if err := json.Unmarshal(change.Payload, &out); err != nil {
				continue
			}

			if out.Kind == "human:move" {
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

			if out.RoomID != "" {
				return out.RoomID, true
			}
		}
	}

	return "", false
}

// ===== Тесты для человека =====

// TestHuman_StartsInRoomWithoutDoors проверяет запуск и движение человека в комнате без дверей.
func TestHuman_StartsInRoomWithoutDoors(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-isolated-room"
	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorOneRoom(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":1.0,"y":1.0,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	step := tick(t, conn, reqID, 1, []api.EventDTO{
		humanMoveInput(t, "human_1", 2.0, 2.0),
	})
	x, y, found := humanPositionFrom([]api.SimulationStepPayload{step}, "human_1")
	if !found || x != 2.0 || y != 2.0 {
		t.Fatalf("human should move inside isolated room, got position (%.2f, %.2f), found=%v", x, y, found)
	}

	for _, change := range step.StateChanges {
		if change.EntityID != "human_1" {
			continue
		}
		var output struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(change.Payload, &output); err != nil {
			t.Fatalf("unmarshal human state: %v", err)
		}
		if output.Kind != "human:move" {
			t.Fatalf("expected human:move state kind, got %q", output.Kind)
		}
		return
	}
	t.Fatal("human state change not found")
}

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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		humanMoveInput(t, "human_1", 2.5, 2.5),
	}))

	x, y, found := humanPositionFrom(steps, "human_1")
	if !found {
		t.Fatal("no position found for human_1")
	}

	if x != 2.5 || y != 2.5 {
		t.Fatalf("expected position (2.5, 2.5), got (%.2f, %.2f)", x, y)
	}
}

// TestHuman_RouteAdvancesOnBackendTicks проверяет, что frontend передает только
// точки маршрута, а промежуточные шаги рассчитывает backend-процесс человека.
func TestHuman_RouteAdvancesOnBackendTicks(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)
	const reqID = "sim-human-route"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorOneRoom(t),
		Devices: []api.EntityDTO{{
			ID:   "human_1",
			Type: "human",
			Info: json.RawMessage(`{"id":"human_1","x":1.0,"y":1.0,"roomID":"room_1","routeStep":0.5}`),
		}},
		Scenarios: []api.ScenarioDTO{},
	})

	routePayload, err := json.Marshal(map[string]any{
		"kind":   "human:route",
		"action": "start",
		"speed":  5,
		"route":  []map[string]float64{{"x": 2, "y": 1}, {"x": 3, "y": 1}},
	})
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}

	steps := []api.SimulationStepPayload{
		tick(t, conn, reqID, 1, []api.EventDTO{{EntityID: "human_1", Payload: routePayload}}),
		tick(t, conn, reqID, 2, nil),
		tick(t, conn, reqID, 3, nil),
		tick(t, conn, reqID, 4, nil),
		tick(t, conn, reqID, 5, nil),
		tick(t, conn, reqID, 6, nil),
	}
	x, y, found := humanPositionFrom(steps, "human_1")
	if !found || math.Abs(x-3) > 1e-9 || math.Abs(y-1) > 1e-9 {
		t.Fatalf("backend route ended at (%.2f, %.2f), found=%v; want (3, 1)", x, y, found)
	}
}

// TestHuman_BlockedByWall проверяет стену по room.area без room.walls и немедленный отход от неё.
func TestHuman_BlockedByWall(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-wall"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRoomsWithoutRoomWallRefs(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":2.5,"y":4.0,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	// пытаемся пройти через стену x=5
	blockedStep := tick(t, conn, reqID, 1, []api.EventDTO{
		humanMoveInput(t, "human_1", 7.5, 4.0),
	})

	x, _, found := humanPositionFrom([]api.SimulationStepPayload{blockedStep}, "human_1")
	if !found {
		t.Fatal("no position found for human_1")
	}

	// человек должен остановиться перед стеной x=5, не пройдя в room_2
	if x >= 5.0 {
		t.Fatalf("human should be blocked by wall at x=5, got x=%.2f", x)
	}

	// комната не должна измениться
	roomID, roomFound := humanRoomFrom([]api.SimulationStepPayload{blockedStep}, "human_1")
	if roomFound && roomID != "room_1" {
		t.Fatalf("human should stay in room_1, got %s", roomID)
	}

	retreatStep := tick(t, conn, reqID, 2, []api.EventDTO{
		humanMoveInput(t, "human_1", x-0.5, 4.0),
	})
	retreatX, _, retreatFound := humanPositionFrom([]api.SimulationStepPayload{retreatStep}, "human_1")
	if !retreatFound {
		t.Fatal("no retreat position found for human_1")
	}
	if retreatX >= x {
		t.Fatalf("human should move away from wall in one command: blocked x=%.6f, retreat x=%.6f", x, retreatX)
	}

	alongWallStep := tick(t, conn, reqID, 3, []api.EventDTO{
		humanMoveInput(t, "human_1", retreatX, 1.0),
	})
	alongX, alongY, alongFound := humanPositionFrom([]api.SimulationStepPayload{alongWallStep}, "human_1")
	if !alongFound || math.Abs(alongX-retreatX) > 1e-9 || math.Abs(alongY-1.0) > 1e-9 {
		t.Fatalf("human should move along wall, got (%.6f, %.6f), found=%v", alongX, alongY, alongFound)
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
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		humanMoveInput(t, "human_1", 7.5, 2.5),
	}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))

	x, _, found := humanPositionFrom(steps, "human_1")
	if !found {
		t.Fatal("no position found for human_1")
	}

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

// TestHuman_MoveThroughOffsetDoor проверяет проход через дверь, линия которой
// не совпадает с room.area из-за толщины стены.
func TestHuman_MoveThroughOffsetDoor(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-offset-door"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRoomsWithOffsetDoor(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":4.5,"y":2.5,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	step := tick(t, conn, reqID, 1, []api.EventDTO{
		humanMoveInput(t, "human_1", 5.5, 2.5),
	})

	x, _, found := humanPositionFrom([]api.SimulationStepPayload{step}, "human_1")
	if !found || x <= 5 {
		t.Fatalf("human should pass through offset door, got x=%.2f, found=%v", x, found)
	}

	roomID, roomFound := humanRoomFrom([]api.SimulationStepPayload{step}, "human_1")
	if !roomFound || roomID != "room_2" {
		t.Fatalf("expected room_2 after offset door crossing, got %q, found=%v", roomID, roomFound)
	}
}

// TestHuman_DoorGapDoesNotTrapMovement проверяет, что короткий шаг в полосу
// стены не меняет комнату и не блокирует последующий отход или проход.
func TestHuman_DoorGapDoesNotTrapMovement(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-human-door-gap"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRoomsWithDoorGap(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{"id":"human_1","x":4.8,"y":2.5,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	gapStep := tick(t, conn, reqID, 1, []api.EventDTO{
		humanMoveInput(t, "human_1", 5.0, 2.5),
	})
	gapX, _, found := humanPositionFrom([]api.SimulationStepPayload{gapStep}, "human_1")
	if !found || gapX >= 4.9 {
		t.Fatalf("human should remain inside room_1 before the gap, got x=%.6f, found=%v", gapX, found)
	}
	if roomID, roomFound := humanRoomFrom([]api.SimulationStepPayload{gapStep}, "human_1"); !roomFound || roomID != "room_1" {
		t.Fatalf("human room changed inside door gap: room=%q, found=%v", roomID, roomFound)
	}

	retreatStep := tick(t, conn, reqID, 2, []api.EventDTO{
		humanMoveInput(t, "human_1", 4.7, 2.5),
	})
	retreatX, _, found := humanPositionFrom([]api.SimulationStepPayload{retreatStep}, "human_1")
	if !found || math.Abs(retreatX-4.7) > 1e-9 {
		t.Fatalf("human should retreat from door gap immediately, got x=%.6f, found=%v", retreatX, found)
	}

	crossStep := tick(t, conn, reqID, 3, []api.EventDTO{
		humanMoveInput(t, "human_1", 5.3, 2.5),
	})
	crossX, _, found := humanPositionFrom([]api.SimulationStepPayload{crossStep}, "human_1")
	if !found || math.Abs(crossX-5.3) > 1e-9 {
		t.Fatalf("human should cross the complete doorway, got x=%.6f, found=%v", crossX, found)
	}
	if roomID, roomFound := humanRoomFrom([]api.SimulationStepPayload{crossStep}, "human_1"); !roomFound || roomID != "room_2" {
		t.Fatalf("expected room_2 after crossing door gap, got %q, found=%v", roomID, roomFound)
	}
}

// TestHuman_InteractionWithLamp проверяет взаимодействия человека с лампой
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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		humanInteractionInput(t, "human_1", "lamp_1", map[string]any{
			"kind":    "lamp:state",
			"turn_on": true,
		}),
	}))

	lampState, found := lastStateOf(steps, "lamp_1")
	if !found {
		t.Fatal("no state change found for lamp_1")
	}

	if !lampState {
		t.Fatal("lamp_1 should be ON after human interaction")
	}
	if len(steps[0].TriggeredEdges) != 0 {
		t.Fatalf("temporary human interaction must not create triggered edges: %+v", steps[0].TriggeredEdges)
	}
}

// TestHuman_InteractionThenMove проверяет взаимодействие с лампой и послежующее передвижение.
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

	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		humanInteractionInput(t, "human_1", "lamp_1", map[string]any{
			"kind":    "lamp:state",
			"turn_on": true,
		}),
	}))

	steps = append(steps, tick(t, conn, reqID, 2, []api.EventDTO{
		humanMoveInput(t, "human_1", 3.0, 3.0),
	}))

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

// TestObserver_Sensor_And_Camera проверяет взаимодействие человека с камерой и сенсором,
// у которого есть зона взаимодействия в виде радиуса.
func TestObserver_Sensor_And_Camera(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-observers"

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
				ID:   "radiusMoveSensorWithoutUpdate_1",
				Type: "lamp_with",
				Info: json.RawMessage(`{"id":"radiusMoveSensorWithoutUpdate_1","delay":0.0,"x":3.0,"y":3.0,"radius":2.0,"turn_on":false}`),
			},
			{
				ID:   "camera_1",
				Type: "camera",
				Info: json.RawMessage(`{"id":"camera_1","delay":0.0,"x":0.0,"y":0.0,"radius":5.0,"turn_on":false}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	// тик 1: человек (1,1)
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventDTO{
		humanMoveInput(t, "human_1", 1.0, 1.0),
	}))

	// лампа должна быть OFF (дальность ~2.8 > 2)
	lampState, _ := lastBoolStateOf(steps, "radiusMoveSensorWithoutUpdate_1", "turn_on")
	if lampState {
		t.Fatal("radiusMoveSensorWithoutUpdate should be OFF")
	}

	// камера должна быть ON (дальность ~1.4 < 5)
	cameraState, _ := lastBoolStateOf(steps, "camera_1", "turn_on")
	if !cameraState {
		t.Fatal("camera should be ON")
	}

	steps = nil

	// тик 3: человек в лампе
	steps = append(steps, tick(t, conn, reqID, 4, []api.EventDTO{
		humanMoveInput(t, "human_1", 3.0, 3.0),
	}))

	lampState, _ = lastBoolStateOf(steps, "radiusMoveSensorWithoutUpdate_1", "turn_on")
	if !lampState {
		t.Fatal("lamp should be ON")
	}

	steps = nil

	// тик 5: человек далеко
	steps = append(steps, tick(t, conn, reqID, 7, []api.EventDTO{
		humanMoveInput(t, "human_1", 10.0, 10.0),
	}))

	lampState, _ = lastBoolStateOf(steps, "radiusMoveSensorWithoutUpdate_1", "turn_on")
	if lampState {
		t.Fatal("lamp should be OFF")
	}

	cameraState, _ = lastBoolStateOf(steps, "camera_1", "turn_on")
	if cameraState {
		t.Fatal("camera should be OFF")
	}
}

// TestObserver_CameraInAnotherRoom_DoesNotTrigger проверяет, что, если камера и человек в разных
// комнатах, то радиус действия камеры игонируется и камера не считывает действия человека.
func TestObserver_CameraInAnotherRoom_DoesNotTrigger(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-camera-other-room"

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRooms(t),
		Devices: []api.EntityDTO{
			{
				ID:   "human_1",
				Type: "human",
				Info: json.RawMessage(`{
					"id":"human_1",
					"x":1.0,
					"y":1.0,
					"roomID":"room_1"
				}`),
			},
			{
				ID:   "camera_1",
				Type: "camera",
				Info: json.RawMessage(`{
					"id":"camera_1",
					"delay":0.0,
					"x":6.0,
					"y":1.0,
					"radius":100.0,
					"turn_on":false
				}`),
			},
		},
	})

	steps := []api.SimulationStepPayload{
		tick(t, conn, reqID, 1, []api.EventDTO{
			humanMoveInput(t, "human_1", 1.0, 1.0),
		}),
	}

	cameraState, cameraFound := lastBoolStateOf(
		steps,
		"camera_1",
		"turn_on",
	)

	if cameraFound && cameraState {
		t.Fatal("camera should not trigger for human in another room")
	}
}

// TestFire_SingleRoom проверяет, что incident-блоки огня покрывают все углы комнаты в нужный тик.
func TestFire_SingleRoom(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-fire-single-room"

	floor := api.Floor{
		Meta: struct {
			Units string `json:"units"`
		}{Units: "meters"},
		Walls:   []api.Wall{},
		Doors:   []api.Door{},
		Windows: []api.Window{},
		Rooms: []api.Room{
			{
				ID:   "room_1",
				Name: "Living Room",
				Area: [][2]float64{{0, 0}, {5, 0}, {5, 5}, {0, 5}},
			},
		},
	}
	floorRaw, _ := json.Marshal(floor)

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: floorRaw,
		Devices: []api.EntityDTO{
			{
				ID:   "fire_1",
				Type: entities.TypeFire,
				Info: json.RawMessage(`{"id":"fire_1","x":0.25,"y":0.25,"roomID":"room_1"}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	// Координаты события должны иметь приоритет над намеренно неверной точкой из simulation:start.
	fireStartPayload, _ := json.Marshal(map[string]any{
		"kind": "fire:spread", "turn_on": true,
		"x": 2.5, "y": 2.5, "roomID": "room_1",
	})
	fireInput := api.EventDTO{EntityID: "fire_1", Payload: fireStartPayload}

	corners := [][2]float64{{0, 0}, {5, 0}, {5, 5}, {0, 5}}
	allCornersReached := false

	const finalSpreadTick = 120
	for i := 1; i <= finalSpreadTick; i++ {
		var inputs []api.EventDTO
		if i == 1 {
			inputs = []api.EventDTO{fireInput}
		}

		step := tick(t, conn, reqID, i, inputs)

		for _, change := range step.StateChanges {
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(change.Payload, &raw); err == nil {
				if _, ok := raw["fires"]; ok {
					t.Fatal("fire output should use incidents, not legacy fires")
				}
			}

			var out struct {
				Kind      string `json:"kind"`
				Incidents []struct {
					RoomID string              `json:"roomID"`
					Blocks []incidentTestBlock `json:"blocks"`
				} `json:"incidents"`
			}
			if err := json.Unmarshal(change.Payload, &out); err != nil {
				continue
			}
			if out.Kind != "fire:spread" {
				continue
			}

			for _, zone := range out.Incidents {
				reached := true

				for _, corner := range corners {
					if !incidentBlocksCoverPoint(zone.Blocks, corner) {
						reached = false
						break
					}
				}

				if reached {
					allCornersReached = true

					if i < 90 {
						t.Fatalf("fire reached all corners too early at tick %d", i)
					}
				}
			}
		}
	}

	if !allCornersReached {
		t.Fatal("fire never reached all 4 corners with incident blocks")
	}

	resetPayload, _ := json.Marshal(map[string]any{"kind": "fire:spread", "reset": true})
	resetStep := tick(t, conn, reqID, finalSpreadTick+1, []api.EventDTO{{EntityID: "fire_1", Payload: resetPayload}})
	resetObserved := false
	for _, change := range resetStep.StateChanges {
		var out struct {
			Kind      string            `json:"kind"`
			Incidents []json.RawMessage `json:"incidents"`
		}
		if json.Unmarshal(change.Payload, &out) == nil && out.Kind == "fire:spread" {
			resetObserved = true
			if len(out.Incidents) != 0 {
				t.Fatalf("reset should return an empty incident snapshot, got %d zones", len(out.Incidents))
			}
		}
	}
	if !resetObserved {
		t.Fatal("fire reset snapshot was not emitted")
	}

	restartPayload, _ := json.Marshal(map[string]any{
		"kind": "fire:spread", "turn_on": true,
		"x": 1.5, "y": 1.5, "roomID": "room_1",
	})
	restartStep := tick(t, conn, reqID, finalSpreadTick+2, []api.EventDTO{{EntityID: "fire_1", Payload: restartPayload}})
	restarted := false
	for _, change := range restartStep.StateChanges {
		var out struct {
			Kind      string            `json:"kind"`
			Incidents []json.RawMessage `json:"incidents"`
		}
		if json.Unmarshal(change.Payload, &out) == nil && out.Kind == "fire:spread" && len(out.Incidents) > 0 {
			restarted = true
		}
	}
	if !restarted {
		t.Fatal("fire was not activated again after reset")
	}
}

// TestFire_HumanMovementDoesNotAdvanceSpreadExtraSteps проверяет, что human:move является
// входом текущего simulation:tick и не добавляет incident дополнительный BFS-шаг.
func TestFire_HumanMovementDoesNotAdvanceSpreadExtraSteps(t *testing.T) {
	server := newSimServer(t)
	controlConn := dialSim(t, server)
	movingConn := dialSim(t, server)

	startPayload := api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: mockFloorTwoRooms(t),
		Devices: []api.EntityDTO{
			{
				ID:   "fire_1",
				Type: entities.TypeFire,
				Info: json.RawMessage(`{"id":"fire_1","x":2.5,"y":2.5,"roomID":"room_1"}`),
			},
			{
				ID:   "human_1",
				Type: entities.TypeHuman,
				Info: json.RawMessage(`{"id":"human_1","x":1.0,"y":1.0,"roomID":"room_1"}`),
			},
		},
	}

	const controlReqID = "sim-fire-control"
	const movingReqID = "sim-fire-moving-human"
	startSim(t, controlConn, controlReqID, startPayload)
	startSim(t, movingConn, movingReqID, startPayload)

	fireStartPayload, err := json.Marshal(map[string]any{
		"kind": "fire:spread", "turn_on": true,
		"x": 2.5, "y": 2.5, "roomID": "room_1",
	})
	if err != nil {
		t.Fatalf("marshal fire input: %v", err)
	}
	fireInput := api.EventDTO{EntityID: "fire_1", Payload: fireStartPayload}

	controlBlocks := 0
	movingBlocks := 0
	for tickN := 1; tickN <= 25; tickN++ {
		var controlInputs []api.EventDTO
		var movingInputs []api.EventDTO
		if tickN == 1 {
			controlInputs = append(controlInputs, fireInput)
			movingInputs = append(movingInputs, fireInput)
		}
		movingInputs = append(movingInputs, humanMoveInput(
			t,
			"human_1",
			1.0+float64(tickN)*0.25,
			1.0,
		))

		controlStep := tick(t, controlConn, controlReqID, tickN, controlInputs)
		movingStep := tick(t, movingConn, movingReqID, tickN, movingInputs)

		nextControlBlocks, controlFound := incidentBlockCount(controlStep, "fire_1", "fire:spread")
		nextMovingBlocks, movingFound := incidentBlockCount(movingStep, "fire_1", "fire:spread")
		if controlFound != movingFound {
			t.Fatalf(
				"tick %d: fire snapshot schedule differs (control=%t, moving=%t)",
				tickN,
				controlFound,
				movingFound,
			)
		}
		if controlFound {
			controlBlocks = nextControlBlocks
			movingBlocks = nextMovingBlocks
		}
		if movingBlocks != controlBlocks {
			t.Fatalf(
				"tick %d: moving human changed fire spread: got %d blocks, want %d",
				tickN,
				movingBlocks,
				controlBlocks,
			)
		}
		if movingStep.SimTime != controlStep.SimTime {
			t.Fatalf(
				"tick %d: moving human changed simulation time: got %v, want %v",
				tickN,
				movingStep.SimTime,
				controlStep.SimTime,
			)
		}
	}
}

// TestFire_SpreadsThroughDoor проверяет что огонь переходит через дверь
// и триггерит датчик пожара в соседней комнате в нужный момент.
func TestFire_SpreadsThroughDoor(t *testing.T) {
	server := newSimServer(t)
	conn := dialSim(t, server)

	const reqID = "sim-fire-spread"

	floorRaw := mockFloorTwoRooms(t)

	startSim(t, conn, reqID, api.SimulationStartPayload{
		DtSim:     1.0,
		Apartment: floorRaw,
		Devices: []api.EntityDTO{
			{
				ID:   "fire_1",
				Type: entities.TypeFire,
				Info: json.RawMessage(`{"id":"fire_1","x":2.5,"y":2.5,"roomID":"room_1"}`),
			},
			{
				ID:   "radiusMoveSensorWithoutUpdate_1",
				Type: entities.TypeFireSensor,
				Info: json.RawMessage(`{"id":"radiusMoveSensorWithoutUpdate_1","x":7.5,"y":2.5,"radius":0.5,"delay":0.0}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	fireStartPayload, _ := json.Marshal(map[string]any{"kind": "fire:spread", "turn_on": true})
	fireInput := api.EventDTO{EntityID: "fire_1", Payload: fireStartPayload}

	sensorTriggeredAt := -1

	var allSteps []api.SimulationStepPayload
	for i := 1; i <= 160; i++ {
		var inputs []api.EventDTO
		if i == 1 {
			inputs = []api.EventDTO{fireInput}
		}

		step := tick(t, conn, reqID, i, inputs)
		allSteps = append(allSteps, step)

		if sensorTriggeredAt == -1 {
			state, found := lastBoolStateOf(allSteps[len(allSteps)-1:], "radiusMoveSensorWithoutUpdate_1", "turn_on")
			if found && state {
				sensorTriggeredAt = i
			}
		}
	}

	if sensorTriggeredAt == -1 {
		t.Fatal("fire sensor in room_2 was never triggered")
	}

	if sensorTriggeredAt <= 1 {
		t.Fatalf("fire sensor triggered too early at tick %d", sensorTriggeredAt)
	}
}
