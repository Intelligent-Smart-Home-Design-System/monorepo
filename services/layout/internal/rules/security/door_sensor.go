package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
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

func (gl *DoorSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
	}

	doorCenter := apartment.GetObjectCenter(frontDoor.Points)

	deviceID := uuid.NewString()
	newDevice := device.NewDevice(deviceID, "door_sensor", "security")
	placement := device.NewPlacement(newDevice, roomID, doorCenter)

	apartmentLayout.Placements[roomID][newDevice.Type] = placement

	return nil
}
