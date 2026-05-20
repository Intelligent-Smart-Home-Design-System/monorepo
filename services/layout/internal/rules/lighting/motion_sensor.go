package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
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

func (r *MotionSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	devicesRooms, err := apartmentStruct.GetRoomsByNames([]string{apartment.RoomPassage, apartment.RoomBathroom})
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, &point.Point{X: 0, Y: 0}, nil)
	}

	return nil
}
