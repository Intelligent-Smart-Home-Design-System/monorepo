package tests_processing

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/devices"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/converter"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/fschuetz04/simgo"
)

// =====Stubs=====
type stubEnginePort struct {
	outChan    chan api.EventDTO
	inChan     chan api.EventDTO
	simulation *simgo.Simulation
	floor      *api.Floor
}

func (s *stubEnginePort) GetOutChan() chan api.EventDTO {
	return s.outChan
}

func (s *stubEnginePort) GetInChan() chan api.EventDTO {
	return s.inChan
}

func (s *stubEnginePort) GetSimulation() *simgo.Simulation {
	return s.simulation
}

func (s *stubEnginePort) GetFloor() *api.Floor {
	return s.floor
}

func (s *stubEnginePort) GetRoomObservers(roomID string) []string {
	return nil
}

func (s *stubEnginePort) NotifyObservers(roomID string, kind string, payload []byte) {
	return
}

func (s *stubEnginePort) DrainInChan() {
	return
}

func (s *stubEnginePort) GetEntity(id string) entities.Entity {
    return nil
}

// =====Tests=====
// проверка парсинга лампы
func TestEntitiesFromDTO_Lamp(t *testing.T) {
	engineStub := &stubEnginePort{}

	lampJSON := []byte(`{"id":"lamp_1","turn_on":false,"delay":1.0,"receivers":[]}`)

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "lamp_1",
			Info: lampJSON,
		},
	}

	entitiesMap, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lampEntity, ok := entitiesMap["lamp_1"].(*devices.Lamp)
	if !ok {
		t.Fatalf("expected *Lamp type, got %T", entitiesMap["lamp_1"])
	}

	if lampEntity.TurnOn != false {
		t.Errorf("expected TurnOn false, got %v", lampEntity.TurnOn)
	}

	if lampEntity.Delay != 1.0 {
		t.Errorf("expected Delay 1.0, got %v", lampEntity.Delay)
	}

	if len(lampEntity.Receivers) != 0 {
		t.Errorf("expected none receivers, got %v", lampEntity.Receivers)
	}
}

// проверка парсинга переключателя
func TestEntitiesFromDTO_Switcher(t *testing.T) {
	engineStub := &stubEnginePort{}

	switcherJSON := []byte(`{"id":"switcher_1","turn_on":true,"delay":0.5,"receivers":["lamp_1"]}`)

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "switcher_1",
			Info: switcherJSON,
		},
	}

	entitiesMap, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	switcherEntity, ok := entitiesMap["switcher_1"].(*devices.Switcher)
	if !ok {
		t.Fatalf("expected *Switcher type, got %T", entitiesMap["switcher_1"])
	}

	if switcherEntity.TurnOn != true {
		t.Errorf("expected TurnOn true, got %v", switcherEntity.TurnOn)
	}

	if switcherEntity.Delay != 0.5 {
		t.Errorf("expected Delay 0.5, got %v", switcherEntity.Delay)
	}

	if len(switcherEntity.Receivers) != 1 || switcherEntity.Receivers[0] != "lamp_1" {
		t.Errorf("expected receivers ['lamp_1'], got %v", switcherEntity.Receivers)
	}
}

// проверка парсинга неизвестного устройства
func TestEntitiesFromDTO_InvalidType(t *testing.T) {
	engineStub := &stubEnginePort{}

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "unknown_1",
			Info: []byte(`{}`),
		},
	}

	_, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err == nil {
		t.Fatal("expected error for invalid entity type, got nil")
	}

	if err != converter.ErrorInvalidFormat {
		t.Fatalf("expected ErrorInvalidFormat, got %v", err)
	}
}

// проверка парсинга поля
func TestParseFloor(t *testing.T) {
	rawJSON := []byte(`{
		"meta": {
			"units": "meters",
			"source": "ChatGPT"
		},
		"walls": [
			{
				"id": "wall_1",
				"points": [[0.0, 0.0], [0.0, 5.0]],
				"width": 0.2
			}
		],
		"doors": [
			{
				"id": "door_1",
				"points": [[2.0, 0.0], [3.0, 0.0]],
				"width": 0.1,
				"rooms": ["room_a", "room_b"],
				"opens_towards_room": "room_b",
				"swing": "left"
			}
		],
		"windows": [
			{
				"id": "window_1",
				"points": [[0.0, 2.0], [0.0, 3.0]],
				"width": 0.15
			}
		],
		"rooms": [
			{
				"id": "room_a",
				"name": "Kitchen",
				"area": [[0.0, 0.0], [5.0, 0.0], [5.0, 5.0], [0.0, 5.0]],
				"walls": ["wall_1"],
				"doors": ["door_1"],
				"windows": ["window_1"]
			},
			{
				"id": "room_b",
				"name": "Bedroom",
				"area": [[5.0, 0.0], [10.0, 0.0], [10.0, 5.0], [5.0, 5.0]],
				"walls": [],
				"doors": ["door_1"],
				"windows": []
			}
		]
	}`)

	simFloor, err := converter.ParseFloor(rawJSON)
	if err != nil {
		t.Fatalf("ParseFloorJ returned unexpected error: %v", err)
	}

	if simFloor.Meta.Units != "meters" {
		t.Errorf("expected units 'meters', got '%s'", simFloor.Meta.Units)
	}

	if len(simFloor.Walls) != 1 {
		t.Fatalf("expected 1 wall, got %d", len(simFloor.Walls))
	}

	if simFloor.Walls[0].ID != "wall_1" || simFloor.Walls[0].Width != 0.2 {
		t.Errorf("unexpected wall data: %+v", simFloor.Walls[0])
	}

	if simFloor.Walls[0].Points[0][1] != 0.0 || simFloor.Walls[0].Points[1][1] != 5.0 {
		t.Errorf("unexpected wall points: %v", simFloor.Walls[0].Points)
	}

	if len(simFloor.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(simFloor.Windows))
	}

	if simFloor.Windows[0].ID != "window_1" || simFloor.Windows[0].Width != 0.15 {
		t.Errorf("unexpected window data: %+v", simFloor.Windows[0])
	}

	if len(simFloor.Rooms) != 2 {
		t.Fatalf("expected 2 rooms, got %d", len(simFloor.Rooms))
	}

	roomA := simFloor.Rooms[0]
	if roomA.ID != "room_a" || roomA.Name != "Kitchen" {
		t.Errorf("unexpected room_a base data: %+v", roomA)
	}

	if len(roomA.Area) != 4 || roomA.Walls[0] != "wall_1" || roomA.Doors[0] != "door_1" {
		t.Errorf("unexpected room_a internal structure: %+v", roomA)
	}

	roomB := simFloor.Rooms[1]
	if roomB.ID != "room_b" || roomB.Name != "Bedroom" {
		t.Errorf("unexpected room_b base data: %+v", roomB)
	}

	if len(simFloor.Doors) != 1 {
		t.Fatalf("expected 1 door, got %d", len(simFloor.Doors))
	}

	door := simFloor.Doors[0]
	if door.ID != "door_1" || door.OpensTowardsRoom != "room_b" || door.Swing != "left" {
		t.Errorf("unexpected door configuration: %+v", door)
	}

	edgesA, existsA := simFloor.Adjacency["room_a"]
	if !existsA || len(edgesA) != 1 {
		t.Fatalf("expected 1 edge for room_a, got exists=%v, count=%d", existsA, len(edgesA))
	}

	if edgesA[0].NeighborRoomID != "room_b" {
		t.Errorf("expected neighbor of room_a to be room_b, got '%s'", edgesA[0].NeighborRoomID)
	}

	if edgesA[0].Door.ID != "door_1" {
		t.Errorf("expected edge to point to door_1, got '%s'", edgesA[0].Door.ID)
	}

	edgesB, existsB := simFloor.Adjacency["room_b"]
	if !existsB || len(edgesB) != 1 {
		t.Fatalf("expected 1 edge for room_b, got exists=%v, count=%d", existsB, len(edgesB))
	}

	if edgesB[0].NeighborRoomID != "room_a" {
		t.Errorf("expected neighbor of room_b to be room_a, got '%s'", edgesB[0].NeighborRoomID)
	}
}
