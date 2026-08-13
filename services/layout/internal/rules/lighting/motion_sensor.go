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

func (r *MotionSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	rooms := zonedAp.GetZonedRoomsByTypes(deviceRooms)

	for _, room := range rooms {
		roomID := room.OrigRoom.ID

		if room.OrigRoom.Type == apartment.RoomPassage {
			p1, p2, err := corridorEndPoints(*room.OrigRoom)
			if err != nil {
				return err
			}

			layout.AddDeviceToLayout(r.Type(), r.track, roomID, p1, nil, nil)
			layout.AddDeviceToLayout(r.Type(), r.track, roomID, p2, nil, nil)
			continue
		}

		sensorPoint, err := cornerNearDoor(zonedAp.OrigAp, *room.OrigRoom)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, sensorPoint, nil, nil)
	}

	return nil
}
