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

	simID      string
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
		// TODO: обработка остальных типов сообщений от клиента
	//case "simulation:start":
	//	c.handleSimulationStart(msg)
	//case "simulation:tick":
	//	c.handleSimulationTick(msg)
	//case "simulation:stop":
	//	c.handleSimulationStop(msg)
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
			if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
				slog.Error("Error while closing connection to client", "error", err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, msg); err != nil {
				slog.Error("Error while writing message to client", "error", err)
				return
			}
		}
	}
}
