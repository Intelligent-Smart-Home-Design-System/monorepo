package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type DoorSensorRule struct {
	track string
}

func NewDoorSensorRule() *DoorSensorRule {
	return &DoorSensorRule{
		track: "security",
	}
}

func (gl *DoorSensorRule) GetType() string {
	return "door_sensor"
}

func (gl *DoorSensorRule) Apply(apartment *entities.Apartment, deviceRooms []string, apartmentLayout *entities.ApartmentLayout) error {
	frontDoor := apartment.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
	}

	doorCenter := entities.GetObjectCenter(frontDoor.Points)

	deviceID := uuid.NewString()
	device := entities.NewDevice(deviceID, "door_sensor", "security")
	placement := entities.NewPlacement(device, roomID, doorCenter)

	apartmentLayout.Placements[roomID][device.Type] = placement

	return nil
}
