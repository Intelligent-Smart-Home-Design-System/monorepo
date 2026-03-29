package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
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

func (r *IlluminationSensorRule) Apply(apartment *entities.Apartment) map[string]map[string]*entities.Placement {
	res := make(map[string]map[string]*entities.Placement)

	deviceRooms := []string{"living", "kitchen"}
	for _, roomType := range deviceRooms {
		rooms := apartment.GetRoomsByType(roomType)
		for _, room := range rooms {
			roomID := room.ID

			if res[roomID] == nil {
				res[roomID] = make(map[string]*entities.Placement)
			}

			deviceID := uuid.NewString()
			device := entities.NewDevice(deviceID, r.GetType(), r.track)
			placement := entities.NewPlacement(device, roomID, &entities.Point{X: 0, Y: 0, Z: 0})
			res[roomID][device.Type] = placement
		}
	}

	return res
}
