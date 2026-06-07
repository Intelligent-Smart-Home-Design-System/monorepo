package api

import (
	"encoding/json"
	"time"
)

// EventDTO структура для входящих/выходящих событий от клиента в simulation:tick
type EventDTO struct {
	EntityID string          `json:"entity_id"`
	Payload  json.RawMessage `json:"payload"`
}

// EntityDTO структура для сущностей (девайсы, люди)
type EntityDTO struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Info json.RawMessage `json:"info"`
}

// ScenarioDTO структура для сценария (приходит от UI в simulation:start),
// описывает устройство (EntityID) и кого оно тригерит (Edges).
type ScenarioDTO struct {
	EntityID string    `json:"id"`
	Edges    []EdgeDTO `json:"edges"`
}

// EdgeDTO структура для связи между устройствами в сценарии
type EdgeDTO struct {
	ToID   string        `json:"to"`
	Action string        `json:"action"`
	Data   []interface{} `json:"data,omitempty"` // доп параметры
}

// Message структура для сообщений симуляции
type Message struct {
	Type    string          `json:"type"`
	Ts      time.Time       `json:"ts"`
	ReqID   string          `json:"reqId,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client -> Server

// HelloPayload структура для приветственного сообщения от клиента
type HelloPayload struct {
	Client   string   `json:"client"`
	Version  string   `json:"version"`
	Features []string `json:"features"`
}

// SimulationStartPayload структура для информации о запуске симуляции
type SimulationStartPayload struct {
	DtSim     float64         `json:"dtSim"`
	Apartment json.RawMessage `json:"apartment"`
	Devices   []EntityDTO     `json:"devices"`
	Scenarios []ScenarioDTO   `json:"scenarios"`
}

// SimulationTickPayload структура для данных каждого тика симуляции
type SimulationTickPayload struct {
	Tick   int        `json:"tick"`
	Inputs []EventDTO `json:"inputs"`
}

// Server -> Client

// HelloAckPayload структура для ответа на приветственное сообщение от сервера
type HelloAckPayload struct {
	Server  string `json:"server"`
	Version string `json:"version"`
}

// SimulationStartedPayload структура для данных при запуске симуляции
type SimulationStartedPayload struct {
	DtSim float64 `json:"dtSim"`
	State string  `json:"state"`
}

// SimulationStepPayload структура для данных каждого тика симуляции
type SimulationStepPayload struct {
	Tick           int         `json:"tick"`
	SimTime        float64     `json:"simTime"`
	StateChanges   []EventDTO  `json:"stateChanges"`
	TriggeredEdges []EdgeDTO   `json:"triggeredEdges"`
	Humans         []EntityDTO `json:"humans"`
}

// SimulationStatusPayload структура для статуса симуляции
type SimulationStatusPayload struct {
	State string  `json:"state"`
	DtSim float64 `json:"dtSim"`
	Tick  int     `json:"tick"`
}

// ErrorPayload структура для ошибок
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// LogEventPayload структура для лога события
type LogEventPayload struct {
	Level   string `json:"level"`
	Device  string `json:"device"`
	Message string `json:"message"`
}

// Floor структура для описания планировки помещения, включая стены, двери, окна и комнаты
type Floor struct {
	Meta struct {
		Units string `json:"units"`
	} `json:"meta"`
	Walls   []Wall   `json:"walls"`
	Doors   []Door   `json:"doors"`
	Windows []Window `json:"windows"`
	Rooms   []Room   `json:"rooms"`
	// Граф смежности: roomID -> список соседних комнат.
	Adjacency map[string][]RoomEdge
}

// Wall структура для описания стены
type Wall struct {
	ID     string        `json:"id"`
	Points [2][2]float64 `json:"points"`
	Width  float64       `json:"width"`
}

// Door структура для описания двери
type Door struct {
	ID               string        `json:"id"`
	Points           [2][2]float64 `json:"points"`
	Width            float64       `json:"width"`
	Rooms            []string      `json:"rooms"`
	OpensTowardsRoom string        `json:"opens_towards_room,omitempty"`
	Swing            string        `json:"swing,omitempty"`
}

// Window структура для описания окна
type Window struct {
	ID     string        `json:"id"`
	Points [2][2]float64 `json:"points"`
	Width  float64       `json:"width"`
}

// Room структура для описания комнаты
type Room struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Area    [][2]float64 `json:"area"`
	Walls   []string     `json:"walls"`
	Doors   []string     `json:"doors"`
	Windows []string     `json:"windows"`
}

// RoomEdge структура для описания связи между комнатами через дверь
type RoomEdge struct {
	NeighborRoomID string
	Door           *Door
}
