package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
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

func (gl *WindowSensorRule) Apply(apartment *entities.Apartment, deviceRooms []string, apartmentLayout *entities.ApartmentLayout) error {
	for _, window := range apartment.Windows {
		if len(window.Rooms) > 1 {
			continue
		}

		roomID := window.Rooms[0]
		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
		}

		windowCenter := entities.GetObjectCenter(window.Points)

		deviceID := uuid.NewString()
		device := entities.NewDevice(deviceID, "window_sensor", "security")
		placement := entities.NewPlacement(device, roomID, windowCenter)

		apartmentLayout.Placements[roomID][device.Type] = placement
	}

	return nil
}
