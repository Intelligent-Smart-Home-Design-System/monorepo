package actors

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

const (
	ActionMove        string = "human:move"
	ActionInteraction string = "human:interaction"
	ActionRoute       string = "human:route"

	defaultRouteIntervalTicks = 7
	minRouteSpeed             = 0.5
	maxRouteSpeed             = 5.0
)

// HumanActionResult интерфейс для результатов действий человека.
type HumanActionResult interface {
	GetStatus() string
}

// Human представляет человека в симуляции. Он может перемещаться по комнатам и взаимодействовать с устройствами.
type Human struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[HumanInData]

	ID        string   `json:"id"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	RoomID    string   `json:"roomID"`
	Receivers []string `json:"receivers"`
	RouteStep float64  `json:"routeStep"`

	route       [][2]float64
	routeIndex  int
	routeSpeed  float64
	routeTicks  int
	routePaused bool
}

// HumanInData описывает входные данные для действий человека. В зависимости от поля Kind, структура может содержать данные для перемещения или взаимодействия.
type HumanInData struct {
	Kind string `json:"kind"`
	To   struct {
		TargetX float64 `json:"x"`
		TargetY float64 `json:"y"`
	} `json:"to"`
	DeviceID      string          `json:"device_id"`
	DevicePayload json.RawMessage `json:"device_payload"`
	Route         []struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"route"`
	Action string  `json:"action"`
	Speed  float64 `json:"speed"`
}

// HumanMoveOutData описывает результат попытки перемещения человека. Содержит конечные координаты, ID комнаты и статус операции.
type HumanMoveOutData struct {
	Kind string `json:"kind"`
	To   struct {
		TargetX float64 `json:"x"`
		TargetY float64 `json:"y"`
	} `json:"to"`
	RoomID string `json:"roomID"`
	Status string `json:"status"`
}

// GetStatus возвращает статус результата передвижения.
func (r HumanMoveOutData) GetStatus() string {
	return r.Status
}

// HumanInteractionOutData описывает результат взаимодействия человека с устройством. Содержит ID устройства, статус операции и тип действия.
type HumanInteractionOutData struct {
	Kind     string `json:"kind"`
	EntityID string `json:"entity_id"`
	Status   string `json:"status"`
}

// HumanRouteOutData сообщает frontend состояние маршрута, рассчитанного backend.
type HumanRouteOutData struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

// GetStatus возвращает текущее состояние выполнения маршрута.
func (r HumanRouteOutData) GetStatus() string {
	return r.Status
}

// GetStatus возвращает статус результата взаимодействия.
func (r HumanInteractionOutData) GetStatus() string {
	return r.Status
}

// NewHuman создает новый экземпляр человека на основе входных данных. Проверяет корректность начальной позиции и комнаты.
func NewHuman(data []byte, engineAPI engine.EnginePort) (*Human, error) {
	var human Human
	if err := json.Unmarshal(data, &human); err != nil {
		return nil, err
	}

	human.enginePort = engineAPI
	human.inStore = *simgo.NewStore[HumanInData](engineAPI.GetSimulation())

	floor := engineAPI.GetFloor()
	roomFound := false
	for _, room := range floor.Rooms {
		if room.ID == human.RoomID {
			roomFound = true
			if !field.PointInRoom(human.X, human.Y, room) {
				return nil, fmt.Errorf("human %s is not inside room %s", human.ID, human.RoomID)
			}

			break
		}
	}
	if !roomFound {
		return nil, fmt.Errorf("human %s has invalid initial room_id %s", human.ID, human.RoomID)
	}
	if human.RouteStep <= 0 {
		human.RouteStep = defaultHumanRouteStep(floor)
	}
	human.routeSpeed = 1

	return &human, nil
}

// HandleInDTO принимает входные данные в виде JSON, парсит их и сохраняет в хранилище для последующей обработки в процессе.
func (h *Human) HandleInDTO(dto []byte) error {
	input := HumanInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	h.inStore.Put(input)

	return nil
}

// HandleOutDTO принимает результат обработки события, оборачивает его в EventDTO и отправляет в движок. Также уведомляет наблюдателей комнаты о перемещении человека.
func (h *Human) HandleOutDTO(dto []byte) {
	outData := api.EventDTO{
		EntityID: h.ID,
		Payload:  dto,
	}
	h.enginePort.GetOutChan() <- outData

	var move HumanMoveOutData
	if err := json.Unmarshal(dto, &move); err != nil || move.Kind != ActionMove {
		return
	}
	movePayload, _ := json.Marshal(map[string]any{
		"kind": "human:move",
		"to":   map[string]float64{"x": h.X, "y": h.Y},
	})
	h.enginePort.NotifyObservers(h.RoomID, "human:move", movePayload)
}

// GetProcessFunc возвращает функцию процесса, которая будет выполняться в симуляции.
func (h *Human) GetProcessFunc() func(process simgo.Process) {
	return h.Process
}

// Process реализует основной цикл обработки событий человека.
func (h *Human) Process(process simgo.Process) {
	for {
		storeElement := h.inStore.Get()
		process.Wait(storeElement.Event)

		inData := storeElement.Item
		outData := h.HandleEvent(inData)
		if outData == nil {
			continue
		}

		dto, err := json.Marshal(outData)
		if err != nil {
			slog.Warn("error marshaling human out data", "error", err, "entity_id", h.ID)
			continue
		}

		h.HandleOutDTO(dto)

		h.enginePort.DrainInChan()
	}
}

// HandleEvent реализует роутинг событий человека.
func (h *Human) HandleEvent(inData HumanInData) HumanActionResult {
	switch inData.Kind {
	case ActionMove:
		return h.handleMove(inData)
	case ActionInteraction:
		return h.HandleInteraction(inData)
	case ActionRoute:
		return h.handleRoute(inData)
	default:
		slog.Warn("unknown human action type",
			"action_type", inData.Kind,
			"human_id", h.ID,
		)

		return HumanInteractionOutData{
			Status: "unknown action type",
		}
	}
}

// handleRoute обрабатывает команды управления маршрутом: запуск, паузу,
// продолжение и остановку. Возвращает новое состояние маршрута для frontend.
func (h *Human) handleRoute(inData HumanInData) HumanActionResult {
	switch inData.Action {
	case "pause":
		if len(h.route) == 0 {
			return HumanRouteOutData{Kind: ActionRoute, Status: "idle"}
		}
		h.routePaused = true
		return HumanRouteOutData{Kind: ActionRoute, Status: "paused"}
	case "resume":
		if len(h.route) == 0 {
			return HumanRouteOutData{Kind: ActionRoute, Status: "idle"}
		}
		h.routePaused = false
		return HumanRouteOutData{Kind: ActionRoute, Status: "running"}
	case "stop":
		h.clearRoute()
		return HumanRouteOutData{Kind: ActionRoute, Status: "stopped"}
	case "", "start":
		if len(inData.Route) == 0 {
			return HumanRouteOutData{Kind: ActionRoute, Status: "empty route"}
		}
		h.route = make([][2]float64, 0, len(inData.Route))
		for _, point := range inData.Route {
			h.route = append(h.route, [2]float64{point.X, point.Y})
		}
		h.routeIndex = 0
		h.routeTicks = 0
		h.routePaused = false
		h.routeSpeed = math.Min(maxRouteSpeed, math.Max(minRouteSpeed, inData.Speed))
		if inData.Speed == 0 {
			h.routeSpeed = 1
		}
		return HumanRouteOutData{Kind: ActionRoute, Status: "running"}
	default:
		return HumanRouteOutData{Kind: ActionRoute, Status: "unknown route action"}
	}
}

// advanceRoute продвигает человека к следующей точке активного маршрута с
// заданной скоростью. Возвращает перемещение или итоговый статус маршрута.
func (h *Human) advanceRoute() HumanActionResult {
	if len(h.route) == 0 || h.routePaused {
		return nil
	}

	h.routeTicks++
	interval := int(math.Round(defaultRouteIntervalTicks / h.routeSpeed))
	if interval < 1 {
		interval = 1
	}
	if h.routeTicks < interval {
		return nil
	}
	h.routeTicks = 0

	target := h.route[h.routeIndex]
	dx := target[0] - h.X
	dy := target[1] - h.Y
	distance := math.Hypot(dx, dy)
	if distance <= movementParamEpsilon {
		h.routeIndex++
		if h.routeIndex >= len(h.route) {
			h.clearRoute()
			return HumanRouteOutData{Kind: ActionRoute, Status: "completed"}
		}
		return nil
	}

	step := math.Min(h.RouteStep, distance)
	input := HumanInData{Kind: ActionMove}
	input.To.TargetX = h.X + dx/distance*step
	input.To.TargetY = h.Y + dy/distance*step
	beforeX, beforeY := h.X, h.Y
	result := h.handleMove(input)
	if math.Hypot(h.X-beforeX, h.Y-beforeY) <= movementParamEpsilon {
		h.clearRoute()
		return HumanRouteOutData{Kind: ActionRoute, Status: "blocked"}
	}

	if math.Hypot(target[0]-h.X, target[1]-h.Y) <= movementParamEpsilon {
		h.routeIndex++
		if h.routeIndex >= len(h.route) {
			h.clearRoute()
			result.Status = "route completed"
		}
	}
	return result
}

// clearRoute удаляет текущий маршрут и сбрасывает связанное с ним состояние.
func (h *Human) clearRoute() {
	h.route = nil
	h.routeIndex = 0
	h.routeTicks = 0
	h.routePaused = false
}

// defaultHumanRouteStep рассчитывает длину шага маршрута относительно
// меньшей стороны плана и возвращает безопасный минимум для вырожденного плана.
func defaultHumanRouteStep(floor *api.Floor) float64 {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, room := range floor.Rooms {
		for _, point := range room.Area {
			minX = math.Min(minX, point[0])
			maxX = math.Max(maxX, point[0])
			minY = math.Min(minY, point[1])
			maxY = math.Max(maxY, point[1])
		}
	}
	if math.IsInf(minX, 1) {
		return 0.015
	}
	return math.Max(math.Min(maxX-minX, maxY-minY)*0.015, movementParamEpsilon*10)
}

// HandleInteraction обрабатывает взаимодействие человека с устройством. Отправляет событие в движок и возвращает результат взаимодействия.
func (h *Human) HandleInteraction(inData HumanInData) HumanInteractionOutData {
	h.enginePort.GetInChan() <- api.EventDTO{
		EntityID: inData.DeviceID,
		Payload:  inData.DevicePayload,
	}

	return HumanInteractionOutData{
		Kind:     inData.Kind,
		EntityID: h.ID,
		Status:   "triggered",
	}
}

// handleMove обрабатывает попытку перемещения человека. Вычисляет новое положение с учётом стен и дверей, обновляет состояние и возвращает результат перемещения.
func (h *Human) handleMove(inData HumanInData) HumanMoveOutData {
	floor := h.enginePort.GetFloor()

	dx := inData.To.TargetX - h.X
	dy := inData.To.TargetY - h.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist == 0 {
		return HumanMoveOutData{
			Kind: inData.Kind,
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
		Kind: inData.Kind,
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
	currentX, currentY := move.x1, move.y1
	currentRoomID := h.RoomID

	for transitions := 0; transitions <= len(floor.Rooms); transitions++ {
		currentRoom := findRoomByID(floor, currentRoomID)
		if currentRoom == nil {
			return currentX, currentY, currentRoomID
		}

		if field.PointInRoom(move.x2, move.y2, *currentRoom) {
			return move.x2, move.y2, currentRoomID
		}

		remainingMove := segment{currentX, currentY, move.x2, move.y2}
		hitT, hitPoint, intersects := roomBoundaryIntersection(remainingMove, currentRoom)
		if !intersects {
			return currentX, currentY, currentRoomID
		}

		nextRoomID, throughDoor := connectedRoomAtDoorPoint(floor, currentRoomID, hitPoint)
		if !throughDoor {
			stopT := math.Max(0, hitT-movementBoundaryInsetRate)
			return currentX + (move.x2-currentX)*stopT,
				currentY + (move.y2-currentY)*stopT,
				currentRoomID
		}

		nextRoom := findRoomByID(floor, nextRoomID)
		if nextRoom == nil {
			return currentX, currentY, currentRoomID
		}
		if field.PointInRoom(move.x2, move.y2, *nextRoom) {
			return move.x2, move.y2, nextRoomID
		}

		// Соседние room.area могут быть разделены полосой толщины стены.
		// Не меняем RoomID, пока движение фактически не достигло полигона
		// соседней комнаты, иначе человек останется между двумя комнатами.
		doorToTarget := segment{hitPoint[0], hitPoint[1], move.x2, move.y2}
		entryT, _, entersNextRoom := roomBoundaryIntersection(doorToTarget, nextRoom)
		if !entersNextRoom {
			stopT := math.Max(0, hitT-movementBoundaryInsetRate)
			return currentX + (move.x2-currentX)*stopT,
				currentY + (move.y2-currentY)*stopT,
				currentRoomID
		}

		advanceT := math.Min(1, entryT+movementBoundaryInsetRate)
		currentX = hitPoint[0] + (move.x2-hitPoint[0])*advanceT
		currentY = hitPoint[1] + (move.y2-hitPoint[1])*advanceT
		currentRoomID = nextRoomID
	}

	return currentX, currentY, currentRoomID
}

// GetID возвращает ID человека.
func (h *Human) GetID() string {
	return h.ID
}

// GetReceiversID возвращает список ID получателей сообщений от человека.
func (h *Human) GetReceiversID() []string {
	return h.Receivers
}

// SetReceivers устанавливает список ID получателей сообщений от человека на основе входных данных.
func (h *Human) SetReceivers(actions []api.EdgeDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ToID
	}

	h.Receivers = receivers
}

// OnTick продвигает активный маршрут ровно один раз за плановый tick движка.
func (h *Human) OnTick() {
	outData := h.advanceRoute()
	if outData == nil {
		return
	}
	dto, err := json.Marshal(outData)
	if err != nil {
		slog.Warn("error marshaling human route tick", "error", err, "entity_id", h.ID)
		return
	}
	h.HandleOutDTO(dto)
}
