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

func (gl *WindowSensorRule) Type() string {
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
			apartmentLayout.Placements[roomID] = make([]*device.Placement, 0)
		}

		windowCenter := apartment.GetObjectCenter(window.Points)

		deviceID := uuid.NewString()
		newDevice := device.NewDevice(deviceID, gl.Type(), gl.track)
		placement := device.NewPlacement(newDevice, &windowCenter, nil)

		apartmentLayout.Placements[roomID] = append(apartmentLayout.Placements[roomID], placement)
	}

	return nil
}
