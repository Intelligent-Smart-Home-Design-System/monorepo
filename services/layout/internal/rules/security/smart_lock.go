package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
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

func (sl *SmartLockRule) Apply(apartment *entities.Apartment, deviceRooms []string, apartmentLayout *entities.ApartmentLayout) error {
	frontDoor := apartment.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door in apartment")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
	}

	doorCenter := entities.GetObjectCenter(frontDoor.Points)

	deviceID := uuid.NewString()
	device := entities.NewDevice(deviceID, "smart_lock", "security")
	placement := entities.NewPlacement(device, roomID, doorCenter)

	apartmentLayout.Placements[roomID][device.Type] = placement

	return nil
}
