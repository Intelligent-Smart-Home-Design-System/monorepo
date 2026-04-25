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

func (sl *SmartLockRule) Type() string {
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
		apartmentLayout.Placements[roomID] = make([]*device.Placement, 0)
	}

	doorCenter := apartment.GetObjectCenter(frontDoor.Points)

	deviceID := uuid.NewString()
	newDevice := device.NewDevice(deviceID, sl.Type(), sl.track)
	placement := device.NewPlacement(newDevice, &doorCenter, nil)

	apartmentLayout.Placements[roomID] = append(apartmentLayout.Placements[roomID], placement)

	return nil
}
