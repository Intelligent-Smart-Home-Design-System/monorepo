package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type MotionSensorRule struct {
	track string
}

func NewMotionSensorRule() *MotionSensorRule {
	return &MotionSensorRule{
		track: "lighting",
	}
}

func (r *MotionSensorRule) GetType() string {
	return "motion_sensor"
}

func (r *MotionSensorRule) Apply(apartment *entities.Apartment) map[string]map[string]*entities.Placement {
	res := make(map[string]map[string]*entities.Placement)

	deviceRooms := []string{"passage", "bathroom"}
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
