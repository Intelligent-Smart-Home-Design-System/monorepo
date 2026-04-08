package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/google/uuid"
)

type SmartLockRule struct {
	track string
}

func NewSmartLockRule() *SmartLockRule {
	return &SmartLockRule{
		track: "security",
	}
}

func (sl *SmartLockRule) GetType() string {
	return "smart_lock"
}

func (sl *SmartLockRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door in apartment")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
	}

	doorCenter := apartment.GetObjectCenter(frontDoor.Points)

	deviceID := uuid.NewString()
	newDevice := device.NewDevice(deviceID, "smart_lock", "security")
	placement := device.NewPlacement(newDevice, roomID, doorCenter)

	apartmentLayout.Placements[roomID][newDevice.Type] = placement

	return nil
}
