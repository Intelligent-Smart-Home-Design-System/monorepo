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
	apartmentStruct := zonedAp.OrigAp
	rooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		place := point.GetCenter(room.Area)
		if place == nil {
			continue
		}

		layout.AddDeviceToLayout(r.Type(), r.track, room.ID, place, nil, nil)
	}

	return nil
}
