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

func (sd *SmartDoorBellRule) Type() string {
	return "smart_doorbell"
}

func (sd *SmartDoorBellRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door in apartment")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make([]*device.Placement, 0)
	}

	deviceID := uuid.NewString()
	newDevice := device.NewDevice(deviceID, sd.Type(), sd.track)
	placement := device.NewPlacement(newDevice, &frontDoor.Points[0], nil)

	apartmentLayout.Placements[roomID] = append(apartmentLayout.Placements[roomID], placement)

	return nil
}
