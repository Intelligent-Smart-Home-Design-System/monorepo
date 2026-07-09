package actors

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

const (
	KindFireSpread          = "fire:spread"
	KindFloodSpread         = "flood:spread"
	KindSmokeSpread         = "smoke:spread"
	defaultIncidentCellSize = 0.5
	blockEpsilon            = 1e-9
)

// Fire обозначает incident-сущность пожара с kind fire:spread.
type Fire = Incident

// Flood обозначает incident-сущность затопления с kind flood:spread.
type Flood = Incident

// Smoke обозначает incident-сущность дыма с kind smoke:spread.
type Smoke = Incident

// Incident хранит общую runtime-модель распространяющегося инцидента.
type Incident struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[IncidentInData]
	eventKind  string

	ID        string   `json:"id"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	RoomID    string   `json:"roomID"`
	CellSize  float64  `json:"cellSize"`
	Receivers []string `json:"receivers"`
	grid      *IncidentGrid
}

// IncidentInData описывает входящее событие активации или тика incident.
type IncidentInData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// IncidentOutData описывает внешний output incident для клиента симуляции.
type IncidentOutData struct {
	Kind      string              `json:"kind"`
	Incidents []*IncidentZoneData `json:"incidents"`
}

// IncidentSpreadPayload описывает payload для observer-ов конкретной комнаты.
type IncidentSpreadPayload struct {
	Kind   string               `json:"kind"`
	RoomID string               `json:"roomID"`
	Blocks []*IncidentBlockData `json:"blocks,omitempty"`
}

// IncidentZoneData группирует блоки incident внутри одной комнаты.
type IncidentZoneData struct {
	RoomID string               `json:"roomID"`
	Blocks []*IncidentBlockData `json:"blocks"`
}

// IncidentBlockData описывает одну активную клетку incident и ее polygon для отображения.
type IncidentBlockData struct {
	ID        string       `json:"id"`
	RoomID    string       `json:"roomID"`
	X         float64      `json:"x"`
	Y         float64      `json:"y"`
	Size      float64      `json:"size"`
	Intensity float64      `json:"intensity"`
	Points    [][2]float64 `json:"points"`
}

// IncidentGrid хранит расчетную BFS-сетку и текущее распространение incident.
type IncidentGrid struct {
	cellSize float64
	floor    *api.Floor
	cells    map[string]*incidentCell
	burning  map[string]bool
	frontier []string
}

// incidentCell описывает внутреннюю клетку сетки по центру и комнате.
type incidentCell struct {
	id     string
	roomID string
	x      float64
	y      float64
}

// incidentNeighbor описывает смещение до соседней клетки в индексах сетки.
type incidentNeighbor struct {
	dx int
	dy int
}

// NewFire создает incident-сущность пожара из JSON и возвращает ее или ошибку парсинга.
func NewFire(data []byte, engineAPI engine.EnginePort) (*Fire, error) {
	return newIncident(data, engineAPI, KindFireSpread)
}

// NewFlood создает incident-сущность затопления из JSON и возвращает ее или ошибку парсинга.
func NewFlood(data []byte, engineAPI engine.EnginePort) (*Flood, error) {
	return newIncident(data, engineAPI, KindFloodSpread)
}

// NewSmoke создает incident-сущность дыма из JSON и возвращает ее или ошибку парсинга.
func NewSmoke(data []byte, engineAPI engine.EnginePort) (*Smoke, error) {
	return newIncident(data, engineAPI, KindSmokeSpread)
}

// newIncident создает общую incident-сущность с заданным event kind и возвращает ее или ошибку парсинга.
func newIncident(data []byte, engineAPI engine.EnginePort, eventKind string) (*Incident, error) {
	var incident Incident
	if err := json.Unmarshal(data, &incident); err != nil {
		return nil, err
	}

	incident.enginePort = engineAPI
	incident.inStore = *simgo.NewStore[IncidentInData](engineAPI.GetSimulation())
	incident.eventKind = eventKind
	if incident.CellSize <= 0 {
		incident.CellSize = defaultIncidentCellSize
	}

	return &incident, nil
}

// HandleInDTO разбирает входящий JSON события, кладет его в simgo-store и возвращает ошибку парсинга.
func (i *Incident) HandleInDTO(dto []byte) error {
	input := IncidentInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	i.inStore.Put(input)

	return nil
}

// HandleOutDTO отправляет исходящий payload наружу и связанным receivers; ничего не возвращает.
func (i *Incident) HandleOutDTO(dto []byte) {
	i.enginePort.GetOutChan() <- api.EventDTO{
		EntityID: i.ID,
		Payload:  dto,
	}

	for _, r := range i.Receivers {
		i.enginePort.GetInChan() <- api.EventDTO{
			EntityID: r,
			Payload:  dto,
		}
	}
}

// GetProcessFunc возвращает функцию simgo-процесса incident-сущности.
func (i *Incident) GetProcessFunc() func(simgo.Process) {
	return i.Process
}

// Process ждет активации incident, затем на каждый тик распространяет сетку и отправляет события; ничего не возвращает.
func (i *Incident) Process(process simgo.Process) {
	for {
		el := i.inStore.Get()
		process.Wait(el.Event)

		if el.Item.TurnOn {
			break
		}
	}

	floor := i.enginePort.GetFloor()
	i.grid = NewIncidentGrid(floor, i.CellSize)
	i.grid.Ignite(i.X, i.Y, i.RoomID)

	for {
		el := i.inStore.Get()
		process.Wait(el.Event)

		i.grid.Step()
		zones := i.grid.Zones()
		i.notifyObservers(zones)

		dto, err := json.Marshal(IncidentOutData{
			Kind:      i.eventKind,
			Incidents: zones,
		})
		if err != nil {
			return
		}

		i.HandleOutDTO(dto)
	}
}

// notifyObservers отправляет zone payload observer-ам комнат, подписанным на kind incident; ничего не возвращает.
func (i *Incident) notifyObservers(zones []*IncidentZoneData) {
	for _, zone := range zones {
		dto, err := json.Marshal(IncidentSpreadPayload{
			Kind:   i.eventKind,
			RoomID: zone.RoomID,
			Blocks: zone.Blocks,
		})
		if err != nil {
			continue
		}

		for _, observerID := range i.enginePort.GetRoomObservers(zone.RoomID) {
			observer := i.enginePort.GetEntity(observerID).(entities.Observer)

			for _, k := range observer.GetObservedKinds() {
				if k == i.eventKind {
					i.enginePort.GetInChan() <- api.EventDTO{
						EntityID: observerID,
						Payload:  dto,
					}

					break
				}
			}
		}
	}
}

// GetID возвращает ID incident-сущности.
func (i *Incident) GetID() string {
	return i.ID
}

// GetReceiversID возвращает ID сущностей, которые получают прямой output incident.
func (i *Incident) GetReceiversID() []string {
	return i.Receivers
}

// SetReceivers сохраняет receivers из dependency edges; ничего не возвращает.
func (i *Incident) SetReceivers(actions []api.EdgeDTO) {
	i.Receivers = make([]string, len(actions))
	for idx, a := range actions {
		i.Receivers[idx] = a.ToID
	}
}

// OnTick помечает incident как tickable-сущность; дополнительной логики не выполняет.
func (i *Incident) OnTick() {}

// NewIncidentGrid создает расчетную BFS-сетку по floor и возвращает готовый grid.
func NewIncidentGrid(floor *api.Floor, cellSize float64) *IncidentGrid {
	if cellSize <= 0 {
		cellSize = defaultIncidentCellSize
	}

	grid := &IncidentGrid{
		cellSize: cellSize,
		floor:    floor,
		cells:    make(map[string]*incidentCell),
		burning:  make(map[string]bool),
	}
	grid.buildCells()

	return grid
}

// Ignite выбирает ближайшую клетку к стартовой точке, активирует ее и задает начальный frontier.
func (g *IncidentGrid) Ignite(x, y float64, roomID string) {
	cell := g.nearestCell(x, y, roomID)
	if cell == nil {
		return
	}

	g.burning[cell.id] = true
	g.frontier = []string{cell.id}
}

// Step выполняет один BFS-шаг распространения incident и обновляет frontier; ничего не возвращает.
func (g *IncidentGrid) Step() {
	if len(g.frontier) == 0 {
		return
	}

	nextSet := make(map[string]bool)
	for _, cellID := range g.frontier {
		cell := g.cells[cellID]
		if cell == nil {
			continue
		}

		for _, neighbor := range g.neighbors(cell) {
			if g.burning[neighbor.id] {
				continue
			}
			if !g.canSpread(cell, neighbor) {
				continue
			}

			g.burning[neighbor.id] = true
			nextSet[neighbor.id] = true
		}
	}

	g.frontier = g.frontier[:0]
	for id := range nextSet {
		g.frontier = append(g.frontier, id)
	}
}

// Zones группирует активные клетки по комнатам и возвращает DTO зон incident.
func (g *IncidentGrid) Zones() []*IncidentZoneData {
	blocksByRoom := make(map[string][]*IncidentBlockData)

	for id := range g.burning {
		cell := g.cells[id]
		if cell == nil {
			continue
		}

		block := g.blockData(cell)
		blocksByRoom[cell.roomID] = append(blocksByRoom[cell.roomID], block)
	}

	zones := make([]*IncidentZoneData, 0, len(blocksByRoom))
	for roomID, blocks := range blocksByRoom {
		zones = append(zones, &IncidentZoneData{
			RoomID: roomID,
			Blocks: blocks,
		})
	}

	return zones
}

// buildCells нарезает комнаты floor на допустимые клетки grid; ничего не возвращает.
func (g *IncidentGrid) buildCells() {
	for _, room := range g.floor.Rooms {
		if len(room.Area) == 0 {
			continue
		}

		minX, maxX, minY, maxY := polygonBounds(room.Area)
		startX := math.Floor(minX/g.cellSize)*g.cellSize + g.cellSize/2
		startY := math.Floor(minY/g.cellSize)*g.cellSize + g.cellSize/2

		for x := startX; x <= maxX; x += g.cellSize {
			for y := startY; y <= maxY; y += g.cellSize {
				if !field.PointInRoom(x, y, room) {
					continue
				}

				cell := &incidentCell{
					id:     g.cellID(room.ID, x, y),
					roomID: room.ID,
					x:      x,
					y:      y,
				}
				g.cells[cell.id] = cell
			}
		}
	}
}

// nearestCell ищет ближайшую клетку в комнате к координатам и возвращает ее или nil.
func (g *IncidentGrid) nearestCell(x, y float64, roomID string) *incidentCell {
	var nearest *incidentCell
	bestDistance := math.Inf(1)

	for _, cell := range g.cells {
		if cell.roomID != roomID {
			continue
		}

		distance := math.Hypot(cell.x-x, cell.y-y)
		if distance < bestDistance {
			bestDistance = distance
			nearest = cell
		}
	}

	return nearest
}

// neighbors находит соседние клетки по четырем направлениям и возвращает существующих соседей.
func (g *IncidentGrid) neighbors(cell *incidentCell) []*incidentCell {
	offsets := []incidentNeighbor{
		{dx: 1, dy: 0},
		{dx: -1, dy: 0},
		{dx: 0, dy: 1},
		{dx: 0, dy: -1},
	}

	neighbors := make([]*incidentCell, 0, len(offsets)*2)
	for _, offset := range offsets {
		for _, roomID := range g.candidateNeighborRooms(cell) {
			id := g.cellID(roomID, cell.x+float64(offset.dx)*g.cellSize, cell.y+float64(offset.dy)*g.cellSize)
			if neighbor := g.cells[id]; neighbor != nil {
				neighbors = append(neighbors, neighbor)
			}
		}
	}

	return neighbors
}

// candidateNeighborRooms возвращает текущую комнату и соседние комнаты, доступные через двери.
func (g *IncidentGrid) candidateNeighborRooms(cell *incidentCell) []string {
	rooms := []string{cell.roomID}
	for _, edge := range g.floor.Adjacency[cell.roomID] {
		if edge.Door != nil {
			rooms = append(rooms, edge.NeighborRoomID)
		}
	}

	return rooms
}

// canSpread проверяет, может ли incident перейти между двумя клетками, и возвращает результат.
func (g *IncidentGrid) canSpread(from, to *incidentCell) bool {
	move := segment{from.x, from.y, to.x, to.y}
	if from.roomID == to.roomID {
		return !g.crossesBlockingWall(from.roomID, move)
	}

	return g.crossesDoorBetween(from.roomID, to.roomID, move)
}

// crossesBlockingWall проверяет пересечение перехода со стенами без дверных проемов и возвращает результат.
func (g *IncidentGrid) crossesBlockingWall(roomID string, move segment) bool {
	room := findRoomByID(g.floor, roomID)
	if room == nil {
		return false
	}

	roomDoors := g.roomDoors(roomID)
	for _, wallID := range room.Walls {
		wall := findWallByID(g.floor, wallID)
		if wall == nil {
			continue
		}

		for _, part := range splitWallByDoors(wall, roomDoors) {
			if _, intersects := intersectSegments(move, part); intersects {
				return true
			}
		}
	}

	return false
}

// crossesDoorBetween проверяет, пересекает ли переход дверь между комнатами, и возвращает результат.
func (g *IncidentGrid) crossesDoorBetween(fromRoomID, toRoomID string, move segment) bool {
	for _, edge := range g.floor.Adjacency[fromRoomID] {
		if edge.NeighborRoomID != toRoomID || edge.Door == nil {
			continue
		}

		door := edge.Door
		doorSeg := segment{
			door.Points[0][0], door.Points[0][1],
			door.Points[1][0], door.Points[1][1],
		}
		if _, intersects := intersectSegments(move, doorSeg); intersects {
			return true
		}
	}

	return false
}

// roomDoors собирает двери комнаты из adjacency и возвращает их список.
func (g *IncidentGrid) roomDoors(roomID string) []*api.Door {
	var doors []*api.Door
	for _, edge := range g.floor.Adjacency[roomID] {
		if edge.Door != nil {
			doors = append(doors, edge.Door)
		}
	}

	return doors
}

// blockData преобразует внутреннюю клетку grid в DTO блока incident и возвращает его.
func (g *IncidentGrid) blockData(cell *incidentCell) *IncidentBlockData {
	return &IncidentBlockData{
		ID:        cell.id,
		RoomID:    cell.roomID,
		X:         cell.x,
		Y:         cell.y,
		Size:      g.cellSize,
		Intensity: 1,
		Points:    g.clippedBlockPoints(cell),
	}
}

// clippedBlockPoints строит polygon блока, обрезает его по стенам комнаты и возвращает точки.
func (g *IncidentGrid) clippedBlockPoints(cell *incidentCell) [][2]float64 {
	half := g.cellSize / 2
	polygon := [][2]float64{
		{cell.x - half, cell.y - half},
		{cell.x + half, cell.y - half},
		{cell.x + half, cell.y + half},
		{cell.x - half, cell.y + half},
	}

	room := findRoomByID(g.floor, cell.roomID)
	if room == nil {
		return polygon
	}

	center := [2]float64{cell.x, cell.y}
	roomDoors := g.roomDoors(cell.roomID)
	for _, wallID := range room.Walls {
		wall := findWallByID(g.floor, wallID)
		if wall == nil {
			continue
		}

		for _, wallPart := range splitWallByDoors(wall, roomDoors) {
			if !polygonIntersectsSegment(polygon, wallPart) {
				continue
			}
			polygon = clipPolygonByLineKeepingPoint(polygon, wallPart, center)
			if len(polygon) == 0 {
				return polygon
			}
		}
	}

	return polygon
}

// cellID строит стабильный ID клетки по комнате и координатам и возвращает строку.
func (g *IncidentGrid) cellID(roomID string, x, y float64) string {
	return roomID + ":" + strconv.FormatInt(int64(math.Round(x/g.cellSize)), 10) + ":" + strconv.FormatInt(int64(math.Round(y/g.cellSize)), 10)
}
