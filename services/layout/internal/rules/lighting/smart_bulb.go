package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type SmartBulbRule struct {
	track string
}

func NewSmartBulbRule() *SmartBulbRule {
	return &SmartBulbRule{
		track: "lighting",
	}
}

func (r *SmartBulbRule) Type() string {
	return "smart_bulb"
}

func (r *SmartBulbRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	ap := zonedAp.OrigAp
	rooms, err := ap.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		roomID := room.ID

		place := point.GetCenter(room.Area)
		if place == nil {
			fallback := point.Point{X: 0, Y: 0}
			place = &fallback
		}

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, place, nil, nil)
	}

	return nil
}
