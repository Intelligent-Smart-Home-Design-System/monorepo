package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/google/uuid"
)

type IlluminationSensorRule struct {
	track string
}

func NewIlluminationSensorRule() *IlluminationSensorRule {
	return &IlluminationSensorRule{
		track: "lighting",
	}
}

func (r *IlluminationSensorRule) GetType() string {
	return "illumination_sensor"
}

func (r *IlluminationSensorRule) Apply(ap *apartment.Apartment) map[string]map[string]*device.Placement {
	res := make(map[string]map[string]*device.Placement)

	deviceRooms := []string{apartment.RoomLiving, apartment.RoomKitchen}
	for _, roomType := range deviceRooms {
		rooms := ap.GetRoomsByType(roomType)
		for _, room := range rooms {
			roomID := room.ID

			if res[roomID] == nil {
				res[roomID] = make(map[string]*device.Placement)
			}

			deviceID := uuid.NewString()
			dev := device.NewDevice(deviceID, r.GetType(), r.track)
			placement := device.NewPlacement(dev, roomID, &device.Point{X: 0, Y: 0, Z: 0})
			res[roomID][dev.Type] = placement
		}
	}

	return res
}
