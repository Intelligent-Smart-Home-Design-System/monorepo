package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/google/uuid"
)

type WindowSensorRule struct {
	track string
}

func NewWindowSensorRule() *WindowSensorRule {
	return &WindowSensorRule{
		track: "security",
	}
}

func (gl *WindowSensorRule) GetType() string {
	return "window_sensor"
}

func (gl *WindowSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	for _, window := range apartmentStruct.Windows {
		if len(window.Rooms) > 1 {
			continue
		}

		roomID := window.Rooms[0]
		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
		}

		windowCenter := apartment.GetObjectCenter(window.Points)

		deviceID := uuid.NewString()
		newDevice := device.NewDevice(deviceID, "window_sensor", "security")
		placement := device.NewPlacement(newDevice, roomID, windowCenter)

		apartmentLayout.Placements[roomID][newDevice.Type] = placement
	}

	return nil
}
