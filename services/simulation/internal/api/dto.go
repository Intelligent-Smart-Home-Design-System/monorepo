package api

import (
	"encoding/json"
	"time"
)

type SimulationService interface {
	Start(reqID string, payload SimulationStartPayload) error
	Tick(reqID string, payload SimulationTickPayload) (*SimulationStepPayload, error)
	Stop(reqID string) error
}

// структуры конфига (по которым передаются данные между сервисами)

// EventInDTO структура для входящих событий от клиента в simulation:tick
type EventInDTO struct {
	Kind     string          `json:"kind"`
	EntityID string          `json:"entityId"`
	Payload  json.RawMessage `json:"payload"`
}

// EventOutDTO структура для обработанных событий симуляции
type EventOutDTO struct {
	Kind     string          `json:"kind"`
	EntityID string          `json:"entityID"`
	Payload  json.RawMessage `json:"payload"`
}

// EntityDTO структура для сущностей (девайсы, люди)
type EntityDTO struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Info json.RawMessage `json:"info"` // парсится позже в converter (метод engine)
}

// CellDTO структура для клетки поля
type CellDTO struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Condition bool `json:"condition"` // true - сгорело; false - дефолт
}

// FieldDTO структура для плана квартиры
type FieldDTO struct {
	Width  int          `json:"width"`
	Height int          `json:"height"`
	Cells  [][]*CellDTO `json:"cells"`
}

// ScenarioDTO структура для сценария (приходит от UI в simulation:start), описывает устройство (EntityID) и
// кого оно тригерит (Edges).
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
type Message struct {
	Type    string          `json:"type"`
	Ts      time.Time       `json:"ts"`
	ReqID   string          `json:"reqId,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client → Server
type HelloPayload struct {
	Client   string   `json:"client"`
	Version  string   `json:"version"`
	Features []string `json:"features"`
}

type SimulationStartPayload struct {
	DtSim     float64       `json:"dtSim"`
	Apartment FieldDTO      `json:"apartment"`
	Devices   []EntityDTO   `json:"devices"`
	Scenarios []ScenarioDTO `json:"scenarios"`
}

type SimulationTickPayload struct {
	Tick   int          `json:"tick"`
	Inputs []EventInDTO `json:"inputs"`
}

// Server → Client
type HelloAckPayload struct {
	Server  string `json:"server"`
	Version string `json:"version"`
}

type SimulationStartedPayload struct {
	DtSim float64 `json:"dtSim"`
	State string  `json:"state"`
}

type SimulationStepPayload struct {
	Tick           int           `json:"tick"`
	SimTime        float64       `json:"simTime"`
	StateChanges   []EventOutDTO `json:"stateChanges"`
	TriggeredEdges []EdgeDTO     `json:"triggeredEdges"`
	Humans         []EntityDTO   `json:"humans"`
}

// SimulationStatusPayload структура для статуса симуляции (Server → Client)
type SimulationStatusPayload struct {
	State string  `json:"state"`
	DtSim float64 `json:"dtSim"`
	Tick  int     `json:"tick"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// LogEventPayload структура для лога события (Server → Client)
type LogEventPayload struct {
	Level   string `json:"level"`
	Device  string `json:"device"`
	Message string `json:"message"`
}
