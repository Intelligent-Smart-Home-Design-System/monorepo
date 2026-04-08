package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type MotionSensorRule struct {
	track string
}

func NewMotionSensorRule() *MotionSensorRule {
	return &MotionSensorRule{
		track: "security",
	}
}

func (gl *MotionSensorRule) GetType() string {
	return "motion_sensor"
}

func (gl *MotionSensorRule) Apply(apartment *entities.Apartment, deviceRooms []string, apartmentLayout *entities.ApartmentLayout) error {
	motionRooms, err := apartment.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range motionRooms {
		roomID := room.ID

		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
		}

		roomCenter, err := room.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		device := entities.NewDevice(deviceID, "motion_sensor", "security")
		placement := entities.NewPlacement(device, roomID, *roomCenter)

		apartmentLayout.Placements[roomID][device.Type] = placement
	}

	return nil
}
