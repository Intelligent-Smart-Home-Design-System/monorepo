package api

import "encoding/json"

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
	X         int `json:"x"`
	Y         int `json:"y"`
	Condition int `json:"condition"` // 1 - сгорело; 0 - дефолт
}

// FieldDTO структура для плана квартиры
type FieldDTO struct {
	Width  int          `json:"width"`
	Height int          `json:"height"`
	Cells  [][]*CellDTO `json:"cells"`
}
