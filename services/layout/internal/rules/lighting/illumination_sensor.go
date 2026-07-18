package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type IlluminationSensorRule struct {
	track string
}

func NewIlluminationSensorRule() *IlluminationSensorRule {
	return &IlluminationSensorRule{
		track: "lighting",
	}
}

func (r *IlluminationSensorRule) Type() string {
	return "illumination_sensor"
}

func (r *IlluminationSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	ap := zonedAp.OrigAp
	devicesRooms, err := ap.GetRoomsByNames([]string{apartment.RoomLiving, apartment.RoomKitchen})
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		place, err := illuminationPoint(ap, *room)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, place, nil, nil)
	}

	return nil
}

func illuminationPoint(apartmentStruct *apartment.Apartment, room apartment.Room) (*point.Point, error) {
	if len(room.Area) == 0 {
		fallback := point.Point{X: 0, Y: 0}
		return &fallback, nil
	}

	roomCenter := point.GetCenter(room.Area)
	if roomCenter == nil {
		fallback := room.Area[0]
		return &fallback, nil
	}

	roomWindows := getRoomWindows(apartmentStruct, room.Name)
	if len(roomWindows) > 0 && len(roomWindows[0].Points) > 0 {
		windowCenter := point.GetObjectCenter(roomWindows[0].Points)
		return cornerNearWindow(room.Area, windowCenter), nil
	}

	return farCornerFromCenter(room.Area, *roomCenter), nil
}
