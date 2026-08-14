package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

type PresenceSensorRule struct {
	track string
}

func NewPresenceSensorRule() *PresenceSensorRule {
	return &PresenceSensorRule{
		track: "lighting",
	}
}

func (r *PresenceSensorRule) Type() string {
	return "presence_sensor"
}

// Ставим датчик присутствия: 2 на концах коридора, в остальных комнатах 1 у двери
func (r *PresenceSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
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

		place, err := cornerNearDoor(zonedAp.OrigAp, *room.OrigRoom)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, place, nil, nil)
	}

	return nil
}
