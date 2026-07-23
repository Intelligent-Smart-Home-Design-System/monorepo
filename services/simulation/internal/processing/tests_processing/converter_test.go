package tests_processing

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/actors"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/devices"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/converter"
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

// TestEntitiesFromDTO_Incidents проверяет создание fire/flood/smoke как общих incident-сущностей.
func TestEntitiesFromDTO_Incidents(t *testing.T) {
	engineStub := &stubEnginePort{simulation: simgo.NewSimulation()}

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "fire_1",
			Type: entities.TypeFire,
			Info: []byte(`{"id":"fire_1","x":1,"y":1,"roomID":"room_1"}`),
		},
		{
			ID:   "flood_1",
			Type: entities.TypeFlood,
			Info: []byte(`{"id":"flood_1","x":1,"y":1,"roomID":"room_1"}`),
		},
		{
			ID:   "smoke_1",
			Type: entities.TypeSmoke,
			Info: []byte(`{"id":"smoke_1","x":1,"y":1,"roomID":"room_1"}`),
		},
	}

	entitiesMap, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := entitiesMap["fire_1"].(*actors.Incident); !ok {
		t.Fatalf("expected fire incident, got %T", entitiesMap["fire_1"])
	}
	if _, ok := entitiesMap["flood_1"].(*actors.Incident); !ok {
		t.Fatalf("expected flood incident, got %T", entitiesMap["flood_1"])
	}
	if _, ok := entitiesMap["smoke_1"].(*actors.Incident); !ok {
		t.Fatalf("expected smoke incident, got %T", entitiesMap["smoke_1"])
	}
}

// TestEntitiesFromDTO_IncidentSensorsObservedKinds проверяет дефолтные подписки specialized incident-сенсоров.
func TestEntitiesFromDTO_IncidentSensorsObservedKinds(t *testing.T) {
	engineStub := &stubEnginePort{simulation: simgo.NewSimulation()}

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "motion_1",
			Type: entities.TypeRadiusMoveSensorWithoutUpdate,
			Info: []byte(`{"id":"motion_1","x":1,"y":1,"radius":1}`),
		},
		{
			ID:   "fire_sensor_1",
			Type: entities.TypeFireSensor,
			Info: []byte(`{"id":"fire_sensor_1","x":1,"y":1,"radius":1}`),
		},
		{
			ID:   "flood_sensor_1",
			Type: entities.TypeFloodSensor,
			Info: []byte(`{"id":"flood_sensor_1","x":1,"y":1,"radius":1}`),
		},
		{
			ID:   "smoke_sensor_1",
			Type: entities.TypeSmokeSensor,
			Info: []byte(`{"id":"smoke_sensor_1","x":1,"y":1,"radius":1}`),
		},
		{
			ID:   "custom_sensor_1",
			Type: entities.TypeRadiusMoveSensorWithoutUpdate,
			Info: []byte(`{"id":"custom_sensor_1","x":1,"y":1,"radius":1,"observedKinds":["smoke:spread"]}`),
		},
	}

	entitiesMap, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertObservedKinds(t, entitiesMap["motion_1"].(*devices.RadiusMoveSensorWithoutUpdate).GetObservedKinds(), []string{"human:move", "device:move"})
	assertObservedKinds(t, entitiesMap["fire_sensor_1"].(*devices.RadiusMoveSensorWithoutUpdate).GetObservedKinds(), []string{actors.KindFireSpread})
	assertObservedKinds(t, entitiesMap["flood_sensor_1"].(*devices.RadiusMoveSensorWithoutUpdate).GetObservedKinds(), []string{actors.KindFloodSpread})
	assertObservedKinds(t, entitiesMap["smoke_sensor_1"].(*devices.RadiusMoveSensorWithoutUpdate).GetObservedKinds(), []string{actors.KindSmokeSpread})
	assertObservedKinds(t, entitiesMap["custom_sensor_1"].(*devices.RadiusMoveSensorWithoutUpdate).GetObservedKinds(), []string{actors.KindSmokeSpread})
}

// assertObservedKinds сравнивает списки kind-ов подписки датчика и завершает тест при несовпадении.
func assertObservedKinds(t *testing.T, actual, expected []string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected observed kinds %v, got %v", expected, actual)
	}
	for idx := range expected {
		if actual[idx] != expected[idx] {
			t.Fatalf("expected observed kinds %v, got %v", expected, actual)
		}
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

// TestParseFloor_BuildsAdjacencyFromRoomReferences проверяет сбор графа смежности,
// когда дверь не содержит полный список комнат, но комнаты ссылаются на нее сами.
func TestParseFloor_BuildsAdjacencyFromRoomReferences(t *testing.T) {
	rawJSON := []byte(`{
		"meta": { "units": "meters" },
		"walls": [
			{ "id": "wall_shared", "points": [[5.0, 0.0], [5.0, 5.0]], "width": 0.2 }
		],
		"doors": [
			{
				"id": "door_1",
				"points": [[5.0, 2.0], [5.0, 3.0]],
				"width": 0.9,
				"rooms": [],
				"opens_towards_room": "room_b"
			}
		],
		"rooms": [
			{
				"id": "room_a",
				"name": "Kitchen",
				"area": [[0.0, 0.0], [5.0, 0.0], [5.0, 5.0], [0.0, 5.0]],
				"walls": ["wall_shared"],
				"doors": ["door_1"],
				"windows": []
			},
			{
				"id": "room_b",
				"name": "Bedroom",
				"area": [[5.0, 0.0], [10.0, 0.0], [10.0, 5.0], [5.0, 5.0]],
				"walls": ["wall_shared"],
				"doors": [],
				"windows": []
			}
		]
	}`)

	simFloor, err := converter.ParseFloor(rawJSON)
	if err != nil {
		t.Fatalf("ParseFloor returned unexpected error: %v", err)
	}

	edgesA, existsA := simFloor.Adjacency["room_a"]
	if !existsA || len(edgesA) != 1 {
		t.Fatalf("expected adjacency for room_a, got exists=%v count=%d", existsA, len(edgesA))
	}
	if edgesA[0].NeighborRoomID != "room_b" {
		t.Fatalf("expected room_a to connect to room_b, got %q", edgesA[0].NeighborRoomID)
	}

	edgesB, existsB := simFloor.Adjacency["room_b"]
	if !existsB || len(edgesB) != 1 {
		t.Fatalf("expected adjacency for room_b, got exists=%v count=%d", existsB, len(edgesB))
	}
	if edgesB[0].NeighborRoomID != "room_a" {
		t.Fatalf("expected room_b to connect to room_a, got %q", edgesB[0].NeighborRoomID)
	}
}
