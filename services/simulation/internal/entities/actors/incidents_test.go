package actors

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/fschuetz04/simgo"
)

type incidentTestEnginePort struct {
	simulation *simgo.Simulation
}

// GetOutChan возвращает новый канал output событий для тестового engine port.
func (e *incidentTestEnginePort) GetOutChan() chan api.EventDTO {
	return make(chan api.EventDTO)
}

// GetInChan возвращает новый канал input событий для тестового engine port.
func (e *incidentTestEnginePort) GetInChan() chan api.EventDTO {
	return make(chan api.EventDTO)
}

// GetSimulation возвращает simgo simulation для создания store в incident-конструкторах.
func (e *incidentTestEnginePort) GetSimulation() *simgo.Simulation {
	return e.simulation
}

// GetFloor возвращает nil, потому что constructor-тесты не используют floor.
func (e *incidentTestEnginePort) GetFloor() *api.Floor {
	return nil
}

// GetRoomObservers возвращает пустой список observer-ов для тестового engine port.
func (e *incidentTestEnginePort) GetRoomObservers(roomID string) []string {
	return nil
}

// NotifyObservers ничего не делает, потому что в этих тестах уведомления не проверяются.
func (e *incidentTestEnginePort) NotifyObservers(roomID string, kind string, payload []byte) {}

// DrainInChan ничего не делает, потому что тестовый engine port не обрабатывает очередь событий.
func (e *incidentTestEnginePort) DrainInChan() {}

// GetEntity возвращает nil, потому что constructor-тесты не обращаются к entity registry.
func (e *incidentTestEnginePort) GetEntity(id string) entities.Entity {
	return nil
}

// TestNewIncidents_HaveSpreadKinds проверяет, что constructors задают правильный kind события.
func TestNewIncidents_HaveSpreadKinds(t *testing.T) {
	enginePort := &incidentTestEnginePort{simulation: simgo.NewSimulation()}
	data := []byte(`{"id":"incident_1","x":1,"y":1,"roomID":"room_1"}`)

	fire, err := NewFire(data, enginePort)
	if err != nil {
		t.Fatalf("new fire: %v", err)
	}
	if fire.eventKind != KindFireSpread {
		t.Fatalf("unexpected fire kind: %q", fire.eventKind)
	}

	flood, err := NewFlood(data, enginePort)
	if err != nil {
		t.Fatalf("new flood: %v", err)
	}
	if flood.eventKind != KindFloodSpread {
		t.Fatalf("unexpected flood kind: %q", flood.eventKind)
	}

	smoke, err := NewSmoke(data, enginePort)
	if err != nil {
		t.Fatalf("new smoke: %v", err)
	}
	if smoke.eventKind != KindSmokeSpread {
		t.Fatalf("unexpected smoke kind: %q", smoke.eventKind)
	}
}

// TestIncidentApplyActivation_UsesEventOrigin проверяет, что первый turn_on заменяет стартовую точку из entity info.
func TestIncidentApplyActivation_UsesEventOrigin(t *testing.T) {
	x, y := 4.25, 2.75
	incident := &Incident{X: 1, Y: 1, RoomID: "old_room"}

	incident.applyActivation(IncidentInData{TurnOn: true, X: &x, Y: &y, RoomID: "room_2"})

	if incident.X != x || incident.Y != y || incident.RoomID != "room_2" {
		t.Fatalf("activation origin was not applied: x=%v y=%v room=%q", incident.X, incident.Y, incident.RoomID)
	}
}

// TestIncidentApplyActivation_KeepsConfiguredFallback проверяет совместимость со старыми событиями без координат.
func TestIncidentApplyActivation_KeepsConfiguredFallback(t *testing.T) {
	incident := &Incident{X: 1, Y: 2, RoomID: "room_1"}

	incident.applyActivation(IncidentInData{TurnOn: true})

	if incident.X != 1 || incident.Y != 2 || incident.RoomID != "room_1" {
		t.Fatalf("configured origin changed: x=%v y=%v room=%q", incident.X, incident.Y, incident.RoomID)
	}
}

// TestIncidentGrids_ShareTopologyButKeepIndependentState проверяет повторное использование
// рассчитанных клеток разными incident без смешивания их burning/frontier.
func TestIncidentGrids_ShareTopologyButKeepIndependentState(t *testing.T) {
	floor := &api.Floor{
		Rooms: []api.Room{{
			ID:   "room_1",
			Area: [][2]float64{{0, 0}, {2, 0}, {2, 2}, {0, 2}},
		}},
	}
	template := NewIncidentGridTemplate(floor, 0.5)
	fireGrid := NewIncidentGridFromTemplate(template)
	smokeGrid := NewIncidentGridFromTemplate(template)

	if fireGrid.IncidentGridTemplate != smokeGrid.IncidentGridTemplate {
		t.Fatal("incident grids do not share their calculated topology")
	}

	fireGrid.Ignite(0.25, 0.25, "room_1")
	if len(fireGrid.burning) != 1 {
		t.Fatalf("fire grid has %d active cells, expected 1", len(fireGrid.burning))
	}
	if len(smokeGrid.burning) != 0 || len(smokeGrid.frontier) != 0 {
		t.Fatal("fire activation leaked into smoke state")
	}

	fireGrid.Reset()
	if len(fireGrid.burning) != 0 || len(fireGrid.frontier) != 0 {
		t.Fatal("reset did not clear incident state")
	}
	if len(template.cells) == 0 {
		t.Fatal("reset removed shared calculated cells")
	}
}

// TestIncidentGrid_DoesNotCrossWallWithoutDoor проверяет, что incident не проходит через общую стену без двери.
func TestIncidentGrid_DoesNotCrossWallWithoutDoor(t *testing.T) {
	floor := &api.Floor{
		Walls: []api.Wall{
			{ID: "wall_shared", Points: [2][2]float64{{5, 0}, {5, 5}}, Width: 0.1},
		},
		Rooms: []api.Room{
			{
				ID:    "room_1",
				Area:  [][2]float64{{0, 0}, {5, 0}, {5, 5}, {0, 5}},
				Walls: []string{"wall_shared"},
			},
			{
				ID:    "room_2",
				Area:  [][2]float64{{5, 0}, {10, 0}, {10, 5}, {5, 5}},
				Walls: []string{"wall_shared"},
			},
		},
		Adjacency: map[string][]api.RoomEdge{},
	}

	grid := NewIncidentGrid(floor, 0.5)
	grid.Ignite(4.5, 2.5, "room_1")
	for i := 0; i < 20; i++ {
		grid.Step()
	}

	for _, zone := range grid.Zones() {
		if zone.RoomID == "room_2" {
			t.Fatal("incident crossed a wall without a door")
		}
	}
}

// TestIncidentGrid_CrossesWallThroughDoor проверяет, что incident проходит в соседнюю комнату через дверь.
func TestIncidentGrid_CrossesWallThroughDoor(t *testing.T) {
	door := api.Door{
		ID:     "door_1",
		Points: [2][2]float64{{5, 2}, {5, 3}},
		Width:  1,
		Rooms:  []string{"room_1", "room_2"},
	}
	floor := &api.Floor{
		Walls: []api.Wall{
			{ID: "wall_shared", Points: [2][2]float64{{5, 0}, {5, 5}}, Width: 0.1},
		},
		Doors: []api.Door{door},
		Rooms: []api.Room{
			{
				ID:    "room_1",
				Area:  [][2]float64{{0, 0}, {5, 0}, {5, 5}, {0, 5}},
				Walls: []string{"wall_shared"},
				Doors: []string{"door_1"},
			},
			{
				ID:    "room_2",
				Area:  [][2]float64{{5, 0}, {10, 0}, {10, 5}, {5, 5}},
				Walls: []string{"wall_shared"},
				Doors: []string{"door_1"},
			},
		},
		Adjacency: map[string][]api.RoomEdge{
			"room_1": {{NeighborRoomID: "room_2", Door: &door}},
			"room_2": {{NeighborRoomID: "room_1", Door: &door}},
		},
	}

	grid := NewIncidentGrid(floor, 0.5)
	grid.Ignite(4.5, 2.5, "room_1")
	for i := 0; i < 4; i++ {
		grid.Step()
	}

	for _, zone := range grid.Zones() {
		if zone.RoomID == "room_2" {
			return
		}
	}
	t.Fatal("incident did not cross through the door")
}

// TestIncidentGrid_DoesNotJumpIntoNeighborRoomFromCenter проверяет, что инцидент не попадает
// в соседнюю комнату сразу после старта из центра, а сначала расходится внутри своей комнаты.
func TestIncidentGrid_DoesNotJumpIntoNeighborRoomFromCenter(t *testing.T) {
	door := api.Door{
		ID:     "door_1",
		Points: [2][2]float64{{10, 4}, {10, 6}},
		Width:  1,
		Rooms:  []string{"room_1", "room_2"},
	}
	floor := &api.Floor{
		Walls: []api.Wall{
			{ID: "wall_shared", Points: [2][2]float64{{10, 0}, {10, 10}}, Width: 0.1},
		},
		Doors: []api.Door{door},
		Rooms: []api.Room{
			{
				ID:    "room_1",
				Area:  [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
				Walls: []string{"wall_shared"},
				Doors: []string{"door_1"},
			},
			{
				ID:    "room_2",
				Area:  [][2]float64{{10, 0}, {20, 0}, {20, 10}, {10, 10}},
				Walls: []string{"wall_shared"},
				Doors: []string{"door_1"},
			},
		},
		Adjacency: map[string][]api.RoomEdge{
			"room_1": {{NeighborRoomID: "room_2", Door: &door}},
			"room_2": {{NeighborRoomID: "room_1", Door: &door}},
		},
	}

	grid := NewIncidentGrid(floor, 1)
	grid.Ignite(5, 5, "room_1")

	for i := 0; i < 4; i++ {
		grid.Step()
		for _, zone := range grid.Zones() {
			if zone.RoomID == "room_2" {
				t.Fatalf("incident jumped into neighbor room too early on step %d", i+1)
			}
		}
	}
}

// TestIncidentGrid_ClipsBlockByWall проверяет, что polygon блока обрезается по стене.
func TestIncidentGrid_ClipsBlockByWall(t *testing.T) {
	floor := &api.Floor{
		Walls: []api.Wall{
			{ID: "wall_right", Points: [2][2]float64{{5, 0}, {5, 5}}, Width: 0.1},
		},
		Rooms: []api.Room{
			{
				ID:    "room_1",
				Area:  [][2]float64{{0, 0}, {5, 0}, {5, 5}, {0, 5}},
				Walls: []string{"wall_right"},
			},
		},
		Adjacency: map[string][]api.RoomEdge{},
	}

	grid := NewIncidentGrid(floor, 0.5)
	cell := &incidentCell{id: "test", roomID: "room_1", x: 4.9, y: 2.5}
	points := grid.clippedBlockPoints(cell)
	if len(points) == 0 {
		t.Fatal("clipped block should not be empty")
	}

	for _, point := range points {
		if point[0] > 5.0+blockEpsilon {
			t.Fatalf("block was not clipped by wall: x=%.3f", point[0])
		}
	}
}

// TestIncidentGrid_SpreadsInAllFourDirections проверяет BFS на отрицательных координатах реального floor.
func TestIncidentGrid_SpreadsInAllFourDirections(t *testing.T) {
	floor := &api.Floor{
		Rooms: []api.Room{{
			ID:   "room_negative",
			Area: [][2]float64{{-2, -2}, {2, -2}, {2, 2}, {-2, 2}},
		}},
		Adjacency: map[string][]api.RoomEdge{},
	}
	grid := NewIncidentGrid(floor, 0.5)
	grid.Ignite(-0.25, -0.25, "room_negative")

	if !grid.Step() {
		t.Fatal("first BFS step did not activate neighbors")
	}

	want := map[[2]float64]bool{
		{-0.75, -0.25}: true,
		{0.25, -0.25}:  true,
		{-0.25, -0.75}: true,
		{-0.25, 0.25}:  true,
	}
	for id := range grid.burning {
		cell := grid.cells[id]
		delete(want, [2]float64{cell.x, cell.y})
	}
	if len(want) != 0 {
		t.Fatalf("BFS did not spread to all directions, missing: %v", want)
	}
}

// TestIncidentGrid_StepStopsAfterFrontierExhausted проверяет отсутствие новых snapshots после заполнения комнаты.
func TestIncidentGrid_StepStopsAfterFrontierExhausted(t *testing.T) {
	floor := &api.Floor{
		Rooms: []api.Room{{
			ID:   "room_small",
			Area: [][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}},
		}},
		Adjacency: map[string][]api.RoomEdge{},
	}
	grid := NewIncidentGrid(floor, 0.5)
	grid.Ignite(0.25, 0.25, "room_small")

	for grid.Step() {
	}
	if grid.Step() {
		t.Fatal("exhausted BFS frontier reported new cells")
	}
}
