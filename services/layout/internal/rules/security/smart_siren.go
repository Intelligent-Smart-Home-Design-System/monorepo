package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type SmartSirenRule struct {
	track string
}

func NewSmartSirenRule() *SmartSirenRule {
	return &SmartSirenRule{
		track: "security",
	}
}

func (gl *SmartSirenRule) GetType() string {
	return "smart_siren"
}

func (gl *SmartSirenRule) Apply(apartment *entities.Apartment, deviceRooms []string, apartmentLayout *entities.ApartmentLayout) error {
	hallRooms, err := apartment.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, hallRoom := range hallRooms {
		roomID := hallRoom.ID
		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
		}

		hallCenter, err := hallRoom.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		device := entities.NewDevice(deviceID, "smart_siren", "security")
		placement := entities.NewPlacement(device, roomID, *hallCenter)

		apartmentLayout.Placements[roomID][device.Type] = placement
	}

	return nil
}
