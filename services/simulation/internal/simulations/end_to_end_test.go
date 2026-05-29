package simulations

import (
	"encoding/json"
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

	payload, err := json.Marshal(map[string]bool{"turn_on": turnOn})
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
				TurnOn bool `json:"turn_on"`
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
		DtSim:     10.0,
		Apartment: mockApartmentRaw(t),
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
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
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
		DtSim:     1.0,
		Apartment: mockApartmentRaw(t),
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

func humanInteractionInput(t *testing.T, EntityID string, devicePayload any) api.EventInDTO {
	t.Helper()

	payload, err := json.Marshal(devicePayload)
	if err != nil {
		t.Fatalf("humanInteractionInput marshal device payload: %v", err)
	}

	return api.EventInDTO{
		EntityID: EntityID,
		Kind:     "human:interaction",
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

// ===== Tests =====

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

// TestHuman_InteractionWithLamp проверяет что человек может включить лампу.
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
				Info: json.RawMessage(`{"id":"lamp_1","delay":0.0,"turned_on":false}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	// человек взаимодействует с лампой — включает её
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{
		humanInteractionInput(t, "lamp_1", map[string]bool{"turn_on": true}),
	}))
	steps = append(steps, tick(t, conn, reqID, 2, nil))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	// проверяем что лампа включилась
	lampState, found := lastStateOf(steps, "lamp_1")
	if !found {
		t.Fatal("no state change found for lamp_1")
	}
	if !lampState {
		t.Fatal("lamp_1 should be ON after human interaction")
	}

	// проверяем что взаимодействие отразилось в stateChanges лампы
	var interactionFound bool
	for _, step := range steps {
		for _, change := range step.StateChanges {
			var out struct {
				TurnOn bool `json:"turn_on"`
			}
			if err := json.Unmarshal(change.Payload, &out); err != nil {
				continue
			}
			if out.TurnOn == true {
				interactionFound = true
			}
		}
	}

	if !interactionFound {
		t.Fatal("expected human_1 interaction with lamp_1 in stateChanges")
	}
}

// TestHuman_InteractionThenMove проверяет что после взаимодействия человек может двигаться.
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
				Info: json.RawMessage(`{"id":"lamp_1","delay":0.0,"turned_on":false}`),
			},
		},
		Scenarios: []api.ScenarioDTO{},
	})

	var steps []api.SimulationStepPayload

	// тик 1: взаимодействие с лампой
	steps = append(steps, tick(t, conn, reqID, 1, []api.EventInDTO{
		humanInteractionInput(t, "lamp_1", map[string]bool{"turn_on": true}),
	}))

	// тик 2: движение
	steps = append(steps, tick(t, conn, reqID, 2, []api.EventInDTO{
		humanMoveInput(t, "human_1", 3.0, 3.0),
	}))
	steps = append(steps, tick(t, conn, reqID, 3, nil))

	// лампа включена
	lampState, found := lastStateOf(steps, "lamp_1")
	if !found {
		t.Fatal("no state change for lamp_1")
	}
	if !lampState {
		t.Fatal("lamp_1 should be ON")
	}

	// человек переместился
	x, y, posFound := humanPositionFrom(steps, "human_1")
	if !posFound {
		t.Fatal("no position found for human_1")
	}
	if x != 3.0 || y != 3.0 {
		t.Fatalf("expected position (3.0, 3.0), got (%.2f, %.2f)", x, y)
	}
}
