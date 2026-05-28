package actors

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

const (
	ActionMove        string = "move"
	ActionInteraction string = "interaction"
)

type HumanActionResult interface {
	GetStatus() string
}

type Human struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[HumanInData]

	ID        string   `json:"id"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	RoomID    string   `json:"roomID"`
	Receivers []string `json:"receivers"`
}

type HumanInData struct {
	Kind string `json:"kind"`
	To   struct {
		TargetX float64 `json:"x"`
		TargetY float64 `json:"y"`
	} `json:"to"`
	EntityID string          `json:"entityID"`
	Payload  json.RawMessage `json:"payload"`
}

type HumanMoveOutData struct {
	To struct {
		TargetX float64 `json:"x"`
		TargetY float64 `json:"y"`
	} `json:"to"`
	RoomID string `json:"roomID"`
	Status string `json:"status"`
}

func (r HumanMoveOutData) GetStatus() string {
	return r.Status
}

type HumanInteractionOutData struct {
	EntityID string `json:"entity_id"`
	Status   string `json:"status"`
}

func (r HumanInteractionOutData) GetStatus() string {
	return r.Status
}

// segment вспомогательная структура для отрезка
type segment struct {
	x1, y1, x2, y2 float64
}

func NewHuman(data []byte, engineAPI engine.EnginePort) (*Human, error) {
	var human Human
	if err := json.Unmarshal(data, &human); err != nil {
		return nil, err
	}

	human.enginePort = engineAPI
	human.inStore = *simgo.NewStore[HumanInData](engineAPI.GetSimulation())

	floor := engineAPI.GetFloor()
	if _, ok := floor.Adjacency[human.RoomID]; !ok {
		return nil, fmt.Errorf("human %s has invalid initial room_id %s", human.ID, human.RoomID)
	}

	for _, room := range floor.Rooms {
		if room.ID == human.RoomID {
			if !field.PointInRoom(human.X, human.Y, room) {
				return nil, fmt.Errorf("human %s is not inside room %s", human.ID, human.RoomID)
			}
			break
		}
	}

	return &human, nil
}

func (h *Human) HandleInDTO(dto []byte) error {
	input := HumanInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	h.inStore.Put(input)
	return nil
}

func (h *Human) HandleOutDTO(dto []byte) {
	outData := api.EventOutDTO{
		EntityID: h.ID,
		Payload:  dto,
	}
	h.enginePort.GetOutChan() <- outData
}

func (h *Human) GetProcessFunc() func(process simgo.Process) {
	return h.Process
}

func (h *Human) Process(process simgo.Process) {
	for {
		storeElement := h.inStore.Get()
		process.Wait(storeElement.Event)

		inData := storeElement.Item
		outData := h.HandleEvent(inData)

		dto, err := json.Marshal(outData)
		if err != nil {
			slog.Warn("error marshaling human out data", "error", err, "entity_id", h.ID)
			continue
		}

		h.HandleOutDTO(dto)
	}
}

// HandleEvent реализует логику движения человека.
// Двигаемся от текущей позиции к цели, проверяя стены и двери.
func (h *Human) HandleEvent(inData HumanInData) HumanActionResult {
	parts := strings.SplitN(inData.Kind, ":", 2)
	if len(parts) != 2 {
		slog.Warn("invalid human action kind", "kind", inData.Kind, "human_id", h.ID)
		return HumanInteractionOutData{Status: "invalid action kind"}
	}
	actionType := parts[1]

	switch actionType {
	case ActionMove:
		return h.handleMove(inData)
	case ActionInteraction:
		return h.HandleInteraction(inData)
	default:
		slog.Warn("unknown human action type",
			"action_type", actionType,
			"human_id", h.ID,
		)
		return HumanInteractionOutData{
			Status: "unknown action type",
		}
	}
}

func (h *Human) HandleInteraction(inData HumanInData) HumanInteractionOutData {
	h.enginePort.GetInChan() <- api.EventInDTO{
		EntityID: inData.EntityID,
		Payload:  inData.Payload,
	}

	return HumanInteractionOutData{
		EntityID: inData.EntityID,
		Status:   "triggered",
	}
}

func (h *Human) handleMove(inData HumanInData) HumanMoveOutData {
	floor := h.enginePort.GetFloor()

	dx := inData.To.TargetX - h.X
	dy := inData.To.TargetY - h.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist == 0 {
		return HumanMoveOutData{
			To: struct {
				TargetX float64 `json:"x"`
				TargetY float64 `json:"y"`
			}{TargetX: h.X, TargetY: h.Y},
			RoomID: h.RoomID,
			Status: "No move",
		}
	}

	move := segment{h.X, h.Y, h.X + dx, h.Y + dy}

	newX, newY, newRoomID := h.resolveMovement(move, floor)

	h.X = newX
	h.Y = newY
	h.RoomID = newRoomID

	return HumanMoveOutData{
		To: struct {
			TargetX float64 `json:"x"`
			TargetY float64 `json:"y"`
		}{TargetX: h.X, TargetY: h.Y},
		RoomID: newRoomID,
		Status: "moved",
	}
}

// resolveMovement находит конечную позицию с учётом стен и дверей.
func (h *Human) resolveMovement(move segment, floor *api.Floor) (float64, float64, string) {
	currentRoom := findRoomByID(floor, h.RoomID)
	if currentRoom == nil {
		return h.X, h.Y, h.RoomID
	}

	closestT := 1.0
	hitWall := false
	newRoomID := h.RoomID

	var roomDoors []*api.Door
	for _, edge := range floor.Adjacency[h.RoomID] {
		if edge.Door != nil {
			roomDoors = append(roomDoors, edge.Door)
		}
	}

	for _, wallID := range currentRoom.Walls {
		wall := findWallByID(floor, wallID)
		if wall == nil {
			continue
		}

		parts := splitWallByDoors(wall, roomDoors)

		for _, part := range parts {
			t, intersects := intersectSegments(move, part)
			if intersects && t < closestT {
				closestT = t
				hitWall = true
			}
		}
	}

	// проверяем двери текущей комнаты через граф смежности
	for _, edge := range floor.Adjacency[h.RoomID] {
		if edge.Door == nil {
			continue
		}

		door := edge.Door
		doorSeg := segment{
			door.Points[0][0], door.Points[0][1],
			door.Points[1][0], door.Points[1][1],
		}

		t, intersects := intersectSegments(move, doorSeg)
		if intersects && t <= closestT {
			closestT = 1.0
			newRoomID = edge.NeighborRoomID
			hitWall = false
		}
	}

	if hitWall {
		stopT := math.Max(0, closestT-0.001)
		return h.X + (move.x2-h.X)*stopT,
			h.Y + (move.y2-h.Y)*stopT,
			h.RoomID
	}

	return h.X + (move.x2-h.X)*closestT,
		h.Y + (move.y2-h.Y)*closestT,
		newRoomID
}

// splitWallByDoors разбивает стену на части исключая дверные проёмы.
func splitWallByDoors(wall *api.Wall, doors []*api.Door) []segment {
	wallStart := wall.Points[0]
	wallEnd := wall.Points[1]

	wallDX := wallEnd[0] - wallStart[0]
	wallDY := wallEnd[1] - wallStart[1]
	wallLenSq := wallDX*wallDX + wallDY*wallDY

	type interval struct {
		t1 float64
		t2 float64
	}
	var gaps []interval

	for _, door := range doors {
		if !doorOnWall(door, wall) {
			continue
		}
		t1 := projectOnSegment(door.Points[0], wallStart, wallEnd, wallLenSq)
		t2 := projectOnSegment(door.Points[1], wallStart, wallEnd, wallLenSq)

		if t1 > t2 {
			t1, t2 = t2, t1
		}

		if t2 > 0 && t1 < 1 {
			gaps = append(gaps, interval{
				math.Max(0, t1),
				math.Min(1, t2),
			})
		}
	}

	if len(gaps) == 0 {
		return []segment{{wallStart[0], wallStart[1], wallEnd[0], wallEnd[1]}}
	}

	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].t1 < gaps[j].t1
	})

	var segments []segment
	prev := 0.0

	for _, gap := range gaps {
		if gap.t1 > prev+1e-10 {
			segments = append(segments, segment{
				wallStart[0] + prev*wallDX,
				wallStart[1] + prev*wallDY,
				wallStart[0] + gap.t1*wallDX,
				wallStart[1] + gap.t1*wallDY,
			})
		}
		if gap.t2 > prev {
			prev = gap.t2
		}
	}

	if prev < 1.0-1e-10 {
		segments = append(segments, segment{
			wallStart[0] + prev*wallDX,
			wallStart[1] + prev*wallDY,
			wallEnd[0],
			wallEnd[1],
		})
	}

	return segments
}

// projectOnSegment проецирует точку p на отрезок и возвращает параметр t ∈ [0..1].
func projectOnSegment(p [2]float64, start, end [2]float64, lenSq float64) float64 {
	dx := end[0] - start[0]
	dy := end[1] - start[1]
	t := ((p[0]-start[0])*dx + (p[1]-start[1])*dy) / lenSq
	return math.Max(0, math.Min(1, t))
}

// findRoomByID находит комнату по ID.
func findRoomByID(floor *api.Floor, roomID string) *api.Room {
	for i, room := range floor.Rooms {
		if room.ID == roomID {
			return &floor.Rooms[i]
		}
	}
	return nil
}

// findWallByID находит стену по ID.
func findWallByID(floor *api.Floor, wallID string) *api.Wall {
	for i, wall := range floor.Walls {
		if wall.ID == wallID {
			return &floor.Walls[i]
		}
	}
	return nil
}

// doorOnWall проверяет что дверь лежит на стене (коллинеарна и перекрывается).
func doorOnWall(door *api.Door, wall *api.Wall) bool {
	wallSeg := segment{
		wall.Points[0][0], wall.Points[0][1],
		wall.Points[1][0], wall.Points[1][1],
	}
	doorSeg := segment{
		door.Points[0][0], door.Points[0][1],
		door.Points[1][0], door.Points[1][1],
	}
	return segmentsCollinearAndOverlap(wallSeg, doorSeg)
}

// intersectSegments находит параметр t пересечения отрезков [0..1].
// t — насколько далеко вдоль первого отрезка находится точка пересечения.
func intersectSegments(a, b segment) (float64, bool) {
	dx1 := a.x2 - a.x1
	dy1 := a.y2 - a.y1
	dx2 := b.x2 - b.x1
	dy2 := b.y2 - b.y1

	denom := dx1*dy2 - dy1*dx2
	if math.Abs(denom) < 1e-10 {
		return 0, false // параллельные отрезки
	}

	// перечение вычисляем
	t := ((b.x1-a.x1)*dy2 - (b.y1-a.y1)*dx2) / denom
	u := ((b.x1-a.x1)*dy1 - (b.y1-a.y1)*dx1) / denom

	if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
		return t, true
	}

	return 0, false
}

// segmentsCollinearAndOverlap проверяет что отрезки коллинеарны и перекрываются.
func segmentsCollinearAndOverlap(a, b segment) bool {
	// проверяем коллинеарность через векторное произведение
	cross := (a.x2-a.x1)*(b.y1-a.y1) - (a.y2-a.y1)*(b.x1-a.x1)
	if math.Abs(cross) > 1e-10 {
		return false
	}

	// проверяем перекрытие проекций на ось X или Y
	aMinX, aMaxX := math.Min(a.x1, a.x2), math.Max(a.x1, a.x2)
	bMinX, bMaxX := math.Min(b.x1, b.x2), math.Max(b.x1, b.x2)
	aMinY, aMaxY := math.Min(a.y1, a.y2), math.Max(a.y1, a.y2)
	bMinY, bMaxY := math.Min(b.y1, b.y2), math.Max(b.y1, b.y2)

	return aMinX <= bMaxX && bMinX <= aMaxX &&
		aMinY <= bMaxY && bMinY <= aMaxY
}

func (h *Human) GetID() string {
	return h.ID
}

func (h *Human) GetReceiversID() []string {
	return h.Receivers
}

func (h *Human) SetReceivers(actions []api.EdgeDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ToID
	}
	h.Receivers = receivers
}
