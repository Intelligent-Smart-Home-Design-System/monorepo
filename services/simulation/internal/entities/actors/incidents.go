package actors

import (
	"encoding/json"
	"math"
	"sort"
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
	Kind   string   `json:"kind"`
	TurnOn bool     `json:"turn_on"`
	Reset  bool     `json:"reset,omitempty"`
	X      *float64 `json:"x,omitempty"`
	Y      *float64 `json:"y,omitempty"`
	RoomID string   `json:"roomID,omitempty"`
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
	ID     string       `json:"id"`
	RoomID string       `json:"roomID"`
	X      float64      `json:"x"`
	Y      float64      `json:"y"`
	Size   float64      `json:"size"`
	Points [][2]float64 `json:"points"`
}

// IncidentGridTemplate хранит общую неизменяемую геометрию BFS-сетки для всех incident симуляции.
type IncidentGridTemplate struct {
	cellSize float64
	floor    *api.Floor
	cells    map[string]*incidentCell
}

// IncidentGrid хранит отдельное состояние распространения incident поверх общей расчетной сетки.
type IncidentGrid struct {
	*IncidentGridTemplate
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
		for {
			el := i.inStore.Get()
			process.Wait(el.Event)

			if el.Item.TurnOn {
				i.start(el.Item)
				i.emitZones(i.grid.Zones())
				break
			}
		}

		for {
			el := i.inStore.Get()
			process.Wait(el.Event)

			if el.Item.Reset {
				i.reset()
				break
			}
			if el.Item.TurnOn {
				i.start(el.Item)
				i.emitZones(i.grid.Zones())
				continue
			}

			if i.grid.Step() {
				i.emitZones(i.grid.Zones())
			}
		}
	}
}

// start очищает состояние incident в общей BFS-сетке и активирует клетку в координатах события.
func (i *Incident) start(input IncidentInData) {
	i.applyActivation(input)
	if i.grid == nil {
		i.grid = NewIncidentGrid(i.enginePort.GetFloor(), i.CellSize)
	} else {
		i.grid.Reset()
	}
	i.grid.Ignite(i.X, i.Y, i.RoomID)
}

// reset очищает incident, выключает затронутые датчики и отправляет пустой snapshot.
func (i *Incident) reset() {
	if i.grid != nil {
		zones := i.grid.Zones()
		for _, zone := range zones {
			zone.Blocks = []*IncidentBlockData{}
		}
		i.notifyObservers(zones)
	}
	if i.grid != nil {
		i.grid.Reset()
	}
	i.emitZones([]*IncidentZoneData{})
}

// SetGridTemplate подключает incident к общей расчетной сетке и создает его независимое BFS-состояние.
func (i *Incident) SetGridTemplate(template *IncidentGridTemplate) {
	i.grid = NewIncidentGridFromTemplate(template)
}

// emitZones уведомляет observers и отправляет полный snapshot incident наружу.
func (i *Incident) emitZones(zones []*IncidentZoneData) {
	i.notifyObservers(zones)
	dto, err := json.Marshal(IncidentOutData{Kind: i.eventKind, Incidents: zones})
	if err != nil {
		return
	}
	i.HandleOutDTO(dto)
}

// applyActivation переносит переданную пользователем стартовую точку в incident перед созданием BFS-сетки.
func (i *Incident) applyActivation(input IncidentInData) {
	if input.X != nil && !math.IsNaN(*input.X) && !math.IsInf(*input.X, 0) {
		i.X = *input.X
	}
	if input.Y != nil && !math.IsNaN(*input.Y) && !math.IsInf(*input.Y, 0) {
		i.Y = *input.Y
	}
	if input.RoomID != "" {
		i.RoomID = input.RoomID
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

// NewIncidentGridTemplate один раз строит общую расчетную BFS-сетку по floor.
func NewIncidentGridTemplate(floor *api.Floor, cellSize float64) *IncidentGridTemplate {
	if cellSize <= 0 {
		cellSize = defaultIncidentCellSize
	}

	template := &IncidentGridTemplate{
		cellSize: cellSize,
		floor:    floor,
		cells:    make(map[string]*incidentCell),
	}
	template.buildCells()

	return template
}

// CellSize возвращает размер клетки общей расчетной сетки.
func (t *IncidentGridTemplate) CellSize() float64 {
	return t.cellSize
}

// NewIncidentGrid создает расчетную сетку и отдельное состояние распространения.
func NewIncidentGrid(floor *api.Floor, cellSize float64) *IncidentGrid {
	return NewIncidentGridFromTemplate(NewIncidentGridTemplate(floor, cellSize))
}

// NewIncidentGridFromTemplate создает независимое BFS-состояние поверх готовой общей сетки.
func NewIncidentGridFromTemplate(template *IncidentGridTemplate) *IncidentGrid {
	return &IncidentGrid{
		IncidentGridTemplate: template,
		burning:              make(map[string]bool),
	}
}

// Reset очищает активные клетки и frontier, сохраняя рассчитанную геометрию сетки.
func (g *IncidentGrid) Reset() {
	clear(g.burning)
	g.frontier = g.frontier[:0]
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

// Step выполняет один BFS-шаг и возвращает true, если были активированы новые клетки.
func (g *IncidentGrid) Step() bool {
	if len(g.frontier) == 0 {
		return false
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
	sort.Strings(g.frontier)

	return len(g.frontier) > 0
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
func (g *IncidentGridTemplate) buildCells() {
	if g.floor == nil {
		return
	}

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
		targetX := cell.x + float64(offset.dx)*g.cellSize
		targetY := cell.y + float64(offset.dy)*g.cellSize
		for _, roomID := range g.candidateNeighborRooms(cell) {
			var neighbor *incidentCell
			if roomID == cell.roomID {
				neighbor = g.cells[g.cellID(roomID, targetX, targetY)]
			} else {
				neighbor = g.nearestCell(targetX, targetY, roomID)
			}
			if neighbor != nil {
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

// crossesDoorBetween проверяет, может ли инцидент перейти из клетки одной комнаты в
// клетку другой комнаты именно через соединяющую их дверь.
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
			if pointToSegmentDistance([2]float64{move.x1, move.y1}, doorSeg) > g.cellSize {
				continue
			}
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
		ID:     cell.id,
		RoomID: cell.roomID,
		X:      cell.x,
		Y:      cell.y,
		Size:   g.cellSize,
		Points: g.clippedBlockPoints(cell),
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

// cellID строит стабильный ID по индексам содержащей координаты клетки.
func (g *IncidentGridTemplate) cellID(roomID string, x, y float64) string {
	column := int64(math.Floor(x / g.cellSize))
	row := int64(math.Floor(y / g.cellSize))
	return roomID + ":" + strconv.FormatInt(column, 10) + ":" + strconv.FormatInt(row, 10)
}
