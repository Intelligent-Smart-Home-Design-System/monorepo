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

// EventInDTO структура для событий симуляции
type EventInDTO struct {
	EntityID string          `json:"entityID"`
	Info     json.RawMessage `json:"info"`
}

// EventOutDTO структура для обработанных событий симуляции
type EventOutDTO struct {
	EntityID string          `json:"entityID"`
	Info     json.RawMessage `json:"info"`
}

// EntityDTO структура для сущностей (девайсы, люди)
type EntityDTO struct {
	ID        string          `json:"id"`
	Receivers []string        `json:"receivers"` // те, кого данная сущность тригерит
	Info      json.RawMessage `json:"info"`      // парсится позже в converter (метод engine)
}

// ActionDTO структура для действия сущностей
type ActionDTO struct {
	ID         string        `json:"id"`
	ActionName string        `json:"action_name"`
	Data       []interface{} `json:"data"` // доп параметры
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

// ScenarioDTO структура для сценария (приходит от UI в simulation:start)
type ScenarioDTO struct {
	ID    string    `json:"id"`
	Edges []EdgeDTO `json:"edges"`
}

// EdgeDTO структура для связи между устройствами в сценарии
type EdgeDTO struct {
	From   string `json:"from,omitempty"`
	To     string `json:"to"`
	Action string `json:"action"`
}

type InputDTO struct {
	Kind    string       `json:"kind"`
	HumanID string       `json:"humanId,omitempty"`
	To      *PositionDTO `json:"to,omitempty"`
}

// PositionDTO структура для координат
type PositionDTO struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
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
	Tick   int        `json:"tick"`
	Inputs []InputDTO `json:"inputs"`
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
	Tick           int         `json:"tick"`
	SimTime        float64     `json:"simTime"`
	StateChanges   []EntityDTO `json:"stateChanges"`
	TriggeredEdges []EdgeDTO   `json:"triggeredEdges"`
	Humans         []EntityDTO `json:"humans"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
