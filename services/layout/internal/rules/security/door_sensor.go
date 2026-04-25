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

func (ds *DoorSensorRule) Type() string {
	return "door_sensor"
}

func (ds *DoorSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make([]*device.Placement, 0)
	}

	doorCenter := apartment.GetObjectCenter(frontDoor.Points)

	deviceID := uuid.NewString()
	newDevice := device.NewDevice(deviceID, ds.Type(), ds.track)
	placement := device.NewPlacement(newDevice, &doorCenter, nil)

	apartmentLayout.Placements[roomID] = append(apartmentLayout.Placements[roomID], placement)

	return nil
}
