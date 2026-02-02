package api

import "encoding/json"

// структуры конфига (по которым передаются данные между сервисами)

// EventInDTO структура для событий симуляции
type EventInDTO struct {
	EntityID string          `json:"entityID"`
	Type     string          `json:"type"`
	Info     json.RawMessage `json:"info"`
}

type EventOutDTO struct {
	EntityID string          `json:"entityID"`
	Type     string          `json:"type"`
	Info     json.RawMessage `json:"info"`
}

// EntityDTO структура для сущностей (девайсы, люди)
type EntityDTO struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Receivers []string        `json:"receivers"` // те, кого данная сущность тригерит
	Info      json.RawMessage `json:"info"`      // парсится позже в converter
}

// CellDTO структура для клетки поля
type CellDTO struct {
	X         int `json:"x"`
	Y         int `json:"y"`
	Condition int `json:"condition"` // пусть 1 - сгорело; 0 - дефолт
}

// FieldDTO структура для плана квартиры
type FieldDTO struct {
	Width  int          `json:"width"`
	Height int          `json:"height"`
	Cells  [][]*CellDTO `json:"cells"`
}
