package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
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
func (r *DecorativeLuminaireRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
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
