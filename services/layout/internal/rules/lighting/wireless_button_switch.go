package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

type WirelessButtonSwitchRule struct {
	track string
}

func NewWirelessButtonSwitchRule() *WirelessButtonSwitchRule {
	return &WirelessButtonSwitchRule{
		track: "lighting",
	}
}

func (r *WirelessButtonSwitchRule) Type() string {
	return "wireless_button_switch"
}

// Ставим по одному выключателю в каждой нужной комнате в угол рядом с дверью
func (r *WirelessButtonSwitchRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	apartmentStruct := zonedAp.OrigAp
	rooms := zonedAp.GetZonedRoomsByTypes(deviceRooms)

	for _, room := range rooms {
		place, err := cornerNearDoor(apartmentStruct, *room.OrigRoom)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, room.OrigRoom.ID, place, nil, nil)
	}

	return nil
}
