package apartment

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/google/uuid"
)

type Layout struct {
	Placements map[string][]*device.Placement `json:"placements"`
	// roomID -> deviceType -> devicePlacement
	// То есть по roomID получаем мапу между
	// типом устройства и его расстановкой.
}

func NewApartmentResult() *Layout {
	return &Layout{Placements: make(map[string][]*device.Placement)}
}

// HasDeviceInRoom проверяет, есть ли данное устройство в комнате
func (al *Layout) HasDeviceInRoom(deviceType, roomID string) bool {
	placements, ok := al.Placements[roomID]
	if !ok {
		return false
	}

	for _, placement := range placements {
		if placement.Device.Type == deviceType {
			return true
		}
	}

	return false
}

// AddDeviceToLayout добавляет устройство в расстановку
func (al *Layout) AddDeviceToLayout(deviceType, deviceTrack, roomID string, position *point.Point, filters filters.DeviceFilter) {
	_, ok := al.Placements[roomID]
	if !ok {
		al.Placements[roomID] = make([]*device.Placement, 0)
	}

	deviceID := uuid.NewString()
	newDevice := device.NewDevice(deviceID, deviceType, deviceTrack)
	placement := device.NewPlacement(newDevice, position, filters)

	al.Placements[roomID] = append(al.Placements[roomID], placement)
}

func (al *Layout) ToJSON() ([]byte, error) {
	return json.MarshalIndent(al, "", "  ")
}

