package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

type SmartDimmerRule struct {
	track string
}

func NewSmartDimmerRule() *SmartDimmerRule {
	return &SmartDimmerRule{
		track: "lighting",
	}
}

func (r *SmartDimmerRule) Type() string {
	return "smart_dimmer"
}

// Ставим по одному диммеру в каждой нужной комнате в угол рядом с дверью
func (r *SmartDimmerRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	apartmentStruct := zonedAp.OrigAp
	rooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		place, err := cornerNearDoor(apartmentStruct, *room)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, room.ID, place, nil)
	}

	return nil
}
