package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
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

func (sl *SmartDoorBellRule) Apply(apartment *entities.Apartment, apartmentLayout *entities.ApartmentLayout) error {
	frontDoor := apartment.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door in apartment")
	}

	roomID := frontDoor.Rooms[0]
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
	}

	deviceID := uuid.NewString()
	device := entities.NewDevice(deviceID, "smart_doorbell", "security")
	placement := entities.NewPlacement(device, roomID, frontDoor.Points[0])

	apartmentLayout.Placements[roomID][device.Type] = placement

	return nil
}
