package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type BuiltInBacklightRule struct {
	track string
}

func NewBuiltInBacklightRule() *BuiltInBacklightRule {
	return &BuiltInBacklightRule{
		track: "lighting",
	}
}

func (r *BuiltInBacklightRule) Type() string {
	return "built_in_backlight"
}

// Ставим встроенную подсветку по одной в комнату в центр комнаты
func (r *BuiltInBacklightRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	rooms := zonedAp.GetZonedRoomsByTypes(deviceRooms)

	for _, room := range rooms {
		place := point.GetCenter(room.OrigRoom.Area)
		if place == nil {
			continue
		}

		layout.AddDeviceToLayout(r.Type(), r.track, room.OrigRoom.ID, place, nil, nil)
	}

	return nil
}
