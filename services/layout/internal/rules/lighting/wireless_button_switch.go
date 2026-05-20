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
func (r *WirelessButtonSwitchRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	rooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		place, err := cornerNearDoor(apartmentStruct, room)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, room.ID, place, nil)
	}

	return nil
}
