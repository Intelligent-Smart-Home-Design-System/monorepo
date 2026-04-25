package device

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type Device struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // Типы должны совпадать с классификацией
	Track string `json:"track"`
}

// Placement представляет расстановку устройства
type Placement struct {
	Device   *Device              `json:"device"`
	Position *point.Point         `json:"position"`
	Filters  *filters.DeviceFilter `json:"filters,omitempty"`
}

func NewDevice(ID, deviceType, trackType string) *Device {
	return &Device{ID: ID, Type: deviceType, Track: trackType}
}

func NewPlacement(device *Device, position *point.Point, filters *filters.DeviceFilter) *Placement {
	return &Placement{Device: device, Position: position, Filters: filters}
}

func (p *Placement) SetFilters(filters *filters.DeviceFilter) {
	p.Filters = filters
}
