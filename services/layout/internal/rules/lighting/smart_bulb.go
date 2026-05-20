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

func (r *SmartBulbRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	rooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		roomID := room.ID

		place, err := room.GetCenter()
		if err != nil {
			fallback := point.Point{X: 0, Y: 0}
			place = &fallback
		}

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, place, nil)
	}

	return nil
}
