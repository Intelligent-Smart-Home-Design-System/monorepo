package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type DecorativeLuminaireRule struct {
	track string
}

func NewDecorativeLuminaireRule() *DecorativeLuminaireRule {
	return &DecorativeLuminaireRule{
		track: "lighting",
	}
}

func (r *DecorativeLuminaireRule) Type() string {
	return "decorative_luminaire"
}

// Ставим декоративный светильник по одному в комнату в центр комнаты
func (r *DecorativeLuminaireRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
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
