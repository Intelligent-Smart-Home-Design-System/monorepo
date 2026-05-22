package actors

import (
	"encoding/json"
	"log/slog"
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

type Human struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[HumanInData]

	ID        string   `json:"id"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	RoomID    string   `json:"room_id"`
	Receivers []string `json:"receivers"`
}

type HumanInData struct {
	TargetX float64 `json:"target_x"`
	TargetY float64 `json:"target_y"`
}

type HumanOutData struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	RoomID string  `json:"room_id"`
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
func (h *Human) HandleEvent(inData HumanInData) HumanOutData {
	floor := h.enginePort.GetFloor()

	dx := inData.TargetX - h.X
	dy := inData.TargetY - h.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist == 0 {
		return HumanOutData{X: h.X, Y: h.Y, RoomID: h.RoomID}
	}

	move := segment{h.X, h.Y, h.X + dx, h.Y + dy}

	newX, newY, newRoomID := h.resolveMovement(move, floor)

	h.X = newX
	h.Y = newY
	h.RoomID = newRoomID

	return HumanOutData{X: h.X, Y: h.Y, RoomID: h.RoomID}
}

// resolveMovement находит конечную позицию с учётом стен и дверей.
// Ищем ближайшее пересечение — с дверью или стеной.
func (h *Human) resolveMovement(move segment, floor api.FloorDTO) (float64, float64, string) {
	currentRoom := h.findRoom(floor)
	if currentRoom == nil {
		return h.X, h.Y, h.RoomID
	}

	closestT := 1.0
	hitWall := false
	newRoomID := h.RoomID

	for _, wall := range currentRoom.Walls {
		if h.wallHasDoor(wall, currentRoom.Doors) {
			continue
		}
		wallSeg := segment{wall.X1, wall.Y1, wall.X2, wall.Y2}
		t, intersects := intersectSegments(move, wallSeg)
		if intersects && t < closestT {
			closestT = t
			hitWall = true
		}
	}

	for _, door := range currentRoom.Doors {
		doorSeg := segment{door.X1, door.Y1, door.X2, door.Y2}
		t, intersects := intersectSegments(move, doorSeg)
		if intersects && t < closestT {
			closestT = t
			hitWall = false
			if h.RoomID == door.FromRoom {
				newRoomID = door.ToRoom
			} else {
				newRoomID = door.FromRoom
			}
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

func (h *Human) findRoom(floor api.FloorDTO) *api.RoomDTO {
	for i, room := range floor.Rooms {
		if room.ID == h.RoomID {
			return &floor.Rooms[i]
		}
	}
	return nil
}

// wallHasDoor проверяет является ли стена дверным проёмом.
func (h *Human) wallHasDoor(wall api.WallDTO, doors []api.DoorDTO) bool {
	for _, door := range doors {
		if segmentsCollinearAndOverlap(
			segment{wall.X1, wall.Y1, wall.X2, wall.Y2},
			segment{door.X1, door.Y1, door.X2, door.Y2},
		) {
			return true
		}
	}
	return false
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
