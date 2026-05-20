package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
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
func (r *BuiltInBacklightRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	rooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		place, err := room.GetCenter()
		if err != nil {
			continue
		}

		layout.AddDeviceToLayout(r.Type(), r.track, room.ID, place, nil)
	}

	return nil
}
