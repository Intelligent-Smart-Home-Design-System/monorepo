package apartment

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
)

type Apartment struct {
	Walls   []Wall   `json:"walls"`
	Doors   []Door   `json:"door"`
	Windows []Window `json:"windows"`
	Rooms   []Room   `json:"rooms"`

	roomsByName map[string][]Room
	wallsByID   map[string]Wall
}

type Wall struct {
	ID     string  `json:"id"`
	Points []point.Point `json:"points"` // начальная и конечная точки
	Width  float64 `json:"width"`
}

type Door struct {
	ID     string   `json:"id"`
	Points []point.Point  `json:"points"`
	Width  float64  `json:"width"`
	Rooms  []string `json:"rooms"` // ID комнат, которые соединяет дверь
}

type Window struct {
	ID     string   `json:"id"`
	Points []point.Point  `json:"points"`
	Height float64  `json:"height"`
	Rooms  []string `json:"rooms"`
}

type Room struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Area    []point.Point  `json:"area"`
	AreaM2  float64  `json:"area_m2"`
	Windows []string `json:"windows"` // ID окон
	Doors   []string `json:"doors"`   // ID дверей
	Walls   []string `json:"walls"`   // ID стен
}

type ApartmentLayout struct {
	Placements map[string]map[string]*device.Placement // roomID -> deviceType -> devicePlacement
	// То есть по roomID получаем мапу между
	// типом устройства и его расстановкой

	// в дальнейшем необходимо будет хранить доп поля в этой структуре (для других модулей)
}

func NewApartmentResult() *ApartmentLayout {
	return &ApartmentLayout{Placements: make(map[string]map[string]*device.Placement)}
}
