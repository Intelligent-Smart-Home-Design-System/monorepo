package config

import "encoding/json"

// структуры конфига (по которым передаются данные между сервисами)

// EventDTO структура для событий симуляции
type EventDTO struct {
	EntityID string `json:"entityID"`
	Cell     Cell   `json:"cell"`
}

// EntityDTO структура для сущностей (девайсы, люди)
type EntityDTO struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Receivers []string        `json:"receivers"` // те, кого данная сущность тригерит
	Info      json.RawMessage `json:"info"`      // парсится позже в decoder
}

// Cell структура для клетки поля
type Cell struct {
	X         int `json:"x"`
	Y         int `json:"y"`
	Condition int `json:"condition"` // пусть 1 - сгорело; 0 - дефолт
}

// FieldDTO структура для плана квартиры
type FieldDTO struct {
	Width  int       `json:"width"`
	Height int       `json:"height"`
	Cells  [][]*Cell `json:"cells"`
}
