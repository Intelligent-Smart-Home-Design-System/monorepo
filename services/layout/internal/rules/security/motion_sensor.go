package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
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

func (gl *MotionSensorRule) Type() string {
	return "motion_sensor"
}

func (gl *MotionSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	motionRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range motionRooms {
		roomID := room.ID

		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
		}

		roomCenter, err := room.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		newDevice := device.NewDevice(deviceID, "motion_sensor", "security")
		placement := device.NewPlacement(newDevice, roomID, roomCenter)

		apartmentLayout.Placements[roomID][newDevice.Type] = placement
	}

	return nil
}
