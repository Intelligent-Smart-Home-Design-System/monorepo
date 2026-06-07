<<<<<<< HEAD
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



func (r *IlluminationSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	ap := zonedAp.OrigAp
	devicesRooms, err := ap.GetRoomsByNames([]string{apartment.RoomLiving, apartment.RoomKitchen})
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, &point.Point{X: 0, Y: 0}, nil, nil)
	}

	return nil
}
=======
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



func (r *IlluminationSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	ap := zonedAp.OrigAp
	devicesRooms, err := ap.GetRoomsByNames([]string{apartment.RoomLiving, apartment.RoomKitchen})
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, &point.Point{X: 0, Y: 0}, nil)
	}

	return nil
}
>>>>>>> 4bf54f8 (hz)
