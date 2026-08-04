package ws

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/client"
	"github.com/gorilla/websocket"
)

const maxMessageSize = 100

// Client предствляуте структуру клиента, подключенного к серверу через WebSocket.
// Он отвечает за чтение сообщений от клиента, роутинг этих сообщений и отправку
// ответов обратно клиенту.
type Client struct {
	connection *websocket.Conn
	manager    *Manager
	egress     chan []byte

	simService client.SimulationService
	sessionMu  sync.Mutex
	sessionID  string
}

// NewClient создает новый экземпляр Client.
func NewClient(conn *websocket.Conn, manager *Manager, simService client.SimulationService) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte, maxMessageSize),
		simService: simService,
	}
}

// ReadMessages запускает цикл чтения сообщений от клиента.
func (c *Client) ReadMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Error while reading message from client", "error", err)
			}

			return
		}

		var msg api.Message

		err = json.Unmarshal(payload, &msg)
		if err != nil {
			slog.Error("Error while unmarshalling message from client", "error", err)
			c.sendError("", "INVALID_MESSAGE", "cannot parse websocket message")
			continue
		}

		c.route(msg)
	}
}

// route обрабатывает входящее сообщение от клиента, вызывая соответствующий обработчик
func (c *Client) route(msg api.Message) {
	switch msg.Type {
	case "hello":
		c.handleHello(msg)
	case "ping":
		c.handlePing(msg)
	case "simulation:start":
		c.handleSimulationStart(msg)
	case "simulation:tick":
		c.handleSimulationTick(msg)
	case "simulation:stop":
		c.handleSimulationStop(msg)
	default:
		c.sendError(msg.ReqID, "UNKNOWN_TYPE", "unknown message type")
	}
}

// handlePing подтверждает прикладной heartbeat, не затрагивая состояние и время симуляции.
func (c *Client) handlePing(msg api.Message) {
	c.send(api.Message{Type: "pong", Ts: time.Now(), ReqID: msg.ReqID})
}

// handleHello обрабатывает сообщение "hello" от клиента, отправляя обратно "hello:ack" с информацией о сервере и версии
func (c *Client) handleHello(msg api.Message) {
	payload, err := json.Marshal(api.HelloAckPayload{
		Server:  "sim-backend",
		Version: "1.0.0",
	})
	if err != nil {
		slog.Error("Error while marshalling hello ack payload", "error", err)
		return
	}

	c.send(api.Message{
		Type:    "hello:ack",
		Ts:      time.Now(),
		ReqID:   msg.ReqID,
		Payload: payload,
	})
}

// handleSimulationStart обрабатывает сообщение "simulation:start" от клиента, пытаясь запустить новую симуляцию с заданными параметрами. В случае успеха отправляет "simulation:started", в случае ошибки - сообщение об ошибке.
func (c *Client) handleSimulationStart(msg api.Message) {
	if !c.requireReqID(msg) {
		return
	}

	var startPayload api.SimulationStartPayload
	if err := json.Unmarshal(msg.Payload, &startPayload); err != nil {
		slog.Error("Error while unmarshalling simulation:start payload", "error", err)
		c.sendError(msg.ReqID, "INVALID_PAYLOAD", "cannot parse simulation:start payload")

		return
	}
	if startPayload.DtSim <= 0 {
		c.sendError(msg.ReqID, "INVALID_PAYLOAD", "dtSim must be greater than zero")
		return
	}

	if err := c.simService.Start(msg.ReqID, startPayload); err != nil {
		slog.Error("Error while starting simulation", "reqID", msg.ReqID, "error", err)
		c.sendError(msg.ReqID, "START_FAILED", err.Error())

		return
	}
	c.attachSession(msg.ReqID)

	payload, err := json.Marshal(api.SimulationStartedPayload{
		DtSim: startPayload.DtSim,
		State: "running",
	})
	if err != nil {
		slog.Error("Error while marshalling simulation:started payload", "error", err)
		return
	}

	c.send(api.Message{
		Type:    "simulation:started",
		Ts:      time.Now(),
		ReqID:   msg.ReqID,
		Payload: payload,
	})
}

// handleSimulationTick обрабатывает сообщение "simulation:tick" от клиента, пытаясь выполнить один шаг симуляции. В случае успеха отправляет "simulation:step" с результатами шага, в случае ошибки - сообщение об ошибке.
func (c *Client) handleSimulationTick(msg api.Message) {
	if !c.requireReqID(msg) {
		return
	}
	if !c.requireAttachedSession(msg.ReqID) {
		return
	}

	var tickPayload api.SimulationTickPayload
	if err := json.Unmarshal(msg.Payload, &tickPayload); err != nil {
		slog.Error("Error while unmarshalling simulation:tick payload", "error", err)
		c.sendError(msg.ReqID, "INVALID_PAYLOAD", "cannot parse simulation:tick payload")

		return
	}
	if tickPayload.Tick <= 0 {
		c.sendError(msg.ReqID, "INVALID_PAYLOAD", "tick must be greater than zero")
		return
	}

	stepResult, err := c.simService.Tick(msg.ReqID, tickPayload)
	if err != nil {
		slog.Error("Error while ticking simulation", "reqID", msg.ReqID, "error", err)
		_ = c.simService.Stop(msg.ReqID)
		c.clearSession(msg.ReqID)
		c.sendError(msg.ReqID, "TICK_FAILED", err.Error())

		return
	}

	payload, err := json.Marshal(stepResult)
	if err != nil {
		slog.Error("Error while marshalling simulation:step payload", "error", err)
		return
	}

	c.send(api.Message{
		Type:    "simulation:step",
		Ts:      time.Now(),
		ReqID:   msg.ReqID,
		Payload: payload,
	})
}

// handleSimulationStop обрабатывает сообщение "simulation:stop" от клиента, пытаясь остановить симуляцию. В случае успеха отправляет "simulation:stopped",
func (c *Client) handleSimulationStop(msg api.Message) {
	if !c.requireReqID(msg) {
		return
	}
	if !c.requireAttachedSession(msg.ReqID) {
		return
	}

	if err := c.simService.Stop(msg.ReqID); err != nil {
		slog.Error("Error while stopping simulation", "reqID", msg.ReqID, "error", err)
		c.sendError(msg.ReqID, "STOP_FAILED", err.Error())

		return
	}
	c.clearSession(msg.ReqID)

	c.send(api.Message{
		Type:  "simulation:stopped",
		Ts:    time.Now(),
		ReqID: msg.ReqID,
	})
}

// attachSession привязывает backend-сессию к WebSocket и останавливает ранее
// привязанную сессию, если клиент запустил новую симуляцию в том же соединении.
func (c *Client) attachSession(reqID string) {
	c.sessionMu.Lock()
	previous := c.sessionID
	c.sessionID = reqID
	c.sessionMu.Unlock()

	if previous != "" && previous != reqID {
		_ = c.simService.Stop(previous)
	}
}

// clearSession удаляет привязку сессии, только если reqID совпадает с текущей
// сессией клиента, чтобы завершение старого запроса не затронуло новую сессию.
func (c *Client) clearSession(reqID string) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if c.sessionID == reqID {
		c.sessionID = ""
	}
}

// takeSession атомарно забирает идентификатор привязанной сессии и очищает его.
// Manager использует результат при разрыве WebSocket, чтобы остановить backend-сессию один раз.
func (c *Client) takeSession() string {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	reqID := c.sessionID
	c.sessionID = ""
	return reqID
}

// requireReqID проверяет обязательный идентификатор симуляционной сессии.
func (c *Client) requireReqID(msg api.Message) bool {
	if strings.TrimSpace(msg.ReqID) != "" {
		return true
	}

	c.sendError("", "MISSING_REQ_ID", "reqId is required")
	return false
}

// requireAttachedSession проверяет, что запрос относится к backend-сессии,
// запущенной через текущее WebSocket-соединение.
func (c *Client) requireAttachedSession(reqID string) bool {
	c.sessionMu.Lock()
	attached := c.sessionID == reqID
	c.sessionMu.Unlock()
	if attached {
		return true
	}

	c.sendError(reqID, "SESSION_NOT_ATTACHED", "websocket is not attached to this simulation")
	return false
}

// send отправляет сообщение клиенту, сериализуя его в JSON и отправляя через канал egress
func (c *Client) send(msg api.Message) {
	data, _ := json.Marshal(msg)
	c.egress <- data
}

// sendError отправляет сообщение об ошибке клиенту с заданным кодом и сообщением
func (c *Client) sendError(reqID, code, message string) {
	payload, err := json.Marshal(api.ErrorPayload{
		Code:    code,
		Message: message,
	})
	if err != nil {
		slog.Error("Error while marshalling error payload", "error", err)
		return
	}

	c.send(api.Message{
		Type:    "error",
		Ts:      time.Now(),
		ReqID:   reqID,
		Payload: payload,
	})
}

// WriteMessages запускает цикл отправки сообщений клиенту, читая их из канала egress. Если канал закрывается, отправляет сообщение о закрытии соединения и завершает работу.
func (c *Client) WriteMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		select {
		case msg, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					slog.Error("Error while closing connection to client", "error", err)
					return
				}

				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, msg); err != nil {
				slog.Error("Error while writing message to client", "error", err)
				return
			}
		}
	}
}
