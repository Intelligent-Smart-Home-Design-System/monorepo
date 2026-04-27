package ws

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/gorilla/websocket"
)

const maxMessageSize = 100

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	egress     chan []byte

	simService api.SimulationService
}

func NewClient(conn *websocket.Conn, manager *Manager, simService api.SimulationService) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte, maxMessageSize),
		simService: simService,
	}
}

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
			return
		}

		c.route(msg)
	}
}

func (c *Client) route(msg api.Message) {
	switch msg.Type {
	case "hello":
		c.handleHello()
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

func (c *Client) handleHello() {
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
		Payload: payload,
	})
}

func (c *Client) handleSimulationStart(msg api.Message) {
	var startPayload api.SimulationStartPayload
	if err := json.Unmarshal(msg.Payload, &startPayload); err != nil {
		slog.Error("Error while unmarshalling simulation:start payload", "error", err)
		c.sendError(msg.ReqID, "INVALID_PAYLOAD", "cannot parse simulation:start payload")
		return
	}
 
	if err := c.simService.Start(msg.ReqID, startPayload); err != nil {
		slog.Error("Error while starting simulation", "reqID", msg.ReqID, "error", err)
		c.sendError(msg.ReqID, "START_FAILED", err.Error())
		return
	}
 
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
 
func (c *Client) handleSimulationTick(msg api.Message) {
	var tickPayload api.SimulationTickPayload
	if err := json.Unmarshal(msg.Payload, &tickPayload); err != nil {
		slog.Error("Error while unmarshalling simulation:tick payload", "error", err)
		c.sendError(msg.ReqID, "INVALID_PAYLOAD", "cannot parse simulation:tick payload")
		return
	}
 
	stepResult, err := c.simService.Tick(msg.ReqID, tickPayload)
	if err != nil {
		slog.Error("Error while ticking simulation", "reqID", msg.ReqID, "error", err)
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
 
func (c *Client) handleSimulationStop(msg api.Message) {
	if err := c.simService.Stop(msg.ReqID); err != nil {
		slog.Error("Error while stopping simulation", "reqID", msg.ReqID, "error", err)
		c.sendError(msg.ReqID, "STOP_FAILED", err.Error())
		return
	}
 
	c.send(api.Message{
		Type:  "simulation:stopped",
		Ts:    time.Now(),
		ReqID: msg.ReqID,
	})
}

func (c *Client) send(msg api.Message) {
	data, _ := json.Marshal(msg)
	c.egress <- data
}

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
