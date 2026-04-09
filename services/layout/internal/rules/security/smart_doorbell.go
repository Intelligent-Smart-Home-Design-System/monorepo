package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/google/uuid"
)

type SmartDoorBellRule struct {
	track string
}

func NewSmartDoorBellRule() *SmartDoorBellRule {
	return &SmartDoorBellRule{
		track: "security",
	}
}

func (sl *SmartDoorBellRule) GetType() string {
	return "smart_doorbell"
}

func (sl *SmartDoorBellRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door in apartment")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
	}

	deviceID := uuid.NewString()
	newDevice := device.NewDevice(deviceID, "smart_doorbell", "security")
	placement := device.NewPlacement(newDevice, roomID, &frontDoor.Points[0])

	apartmentLayout.Placements[roomID][newDevice.Type] = placement

	return nil
}
