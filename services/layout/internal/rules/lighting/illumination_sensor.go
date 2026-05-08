package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type IlluminationSensorRule struct {
	track string
}

func NewIlluminationSensorRule() *IlluminationSensorRule {
	return &IlluminationSensorRule{
		track: "lighting",
	}
}

func (r *IlluminationSensorRule) Type() string {
	return "illumination_sensor"
}

func (r *IlluminationSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	devicesRooms, err := apartmentStruct.GetRoomsByNames([]string{apartment.RoomLiving, apartment.RoomKitchen})
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, &point.Point{X: 0, Y: 0}, nil)
	}

	return nil
}
