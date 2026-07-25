package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type SmartLampRule struct {
	track string
}

func NewSmartLampRule() *SmartLampRule {
	return &SmartLampRule{
		track: "lighting",
	}
}

func (r *SmartLampRule) Type() string {
	return "smart_lamp"
}

func (r *SmartLampRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
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
