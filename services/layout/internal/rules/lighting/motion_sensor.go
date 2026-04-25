package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/google/uuid"
)

type MotionSensorRule struct {
	track string
}

func NewMotionSensorRule() *MotionSensorRule {
	return &MotionSensorRule{
		track: "lighting",
	}
}

func (r *MotionSensorRule) Type() string {
	return "motion_sensor"
}

func (r *MotionSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	devicesRooms, err := apartmentStruct.GetRoomsByNames([]string{apartment.RoomPassage, apartment.RoomBathroom})
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make([]*device.Placement, 0)
		}

		deviceID := uuid.NewString()
		dev := device.NewDevice(deviceID, r.Type(), r.track)
		placement := device.NewPlacement(dev, &point.Point{X: 0, Y: 0}, nil)
		apartmentLayout.Placements[roomID] = append(apartmentLayout.Placements[roomID], placement)
	}

	return nil
}
