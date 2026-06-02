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

func (r *SmartBulbRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	ap := zonedAp.OrigAp
	devicesRooms, err := ap.GetRoomsByNames([]string{apartment.RoomLiving, apartment.RoomBedroom, apartment.RoomKitchen, apartment.RoomPassage, apartment.RoomBathroom})
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, &point.Point{X: 0, Y: 0}, nil, nil)
	}

	return nil
}
