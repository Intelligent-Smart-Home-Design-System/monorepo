package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

type MotionSensorRule struct {
	track string
}

func NewMotionSensorRule() *MotionSensorRule {
	return &MotionSensorRule{track: "lighting"}
}

func (r *MotionSensorRule) Type() string {
	return "motion_sensor"
}

func (r *MotionSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	rooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		roomID := room.ID

		if room.Name == apartment.RoomPassage {
			p1, p2, err := corridorEndPoints(room)
			if err != nil {
				return err
			}

			layout.AddDeviceToLayout(r.Type(), r.track, roomID, p1, nil)
			layout.AddDeviceToLayout(r.Type(), r.track, roomID, p2, nil)
			continue
		}

		sensorPoint, err := cornerNearDoor(apartmentStruct, room)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, sensorPoint, nil)
	}

	return nil
}
