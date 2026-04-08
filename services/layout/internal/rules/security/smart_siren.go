package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
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

func (gl *SmartSirenRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	hallRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, hallRoom := range hallRooms {
		roomID := hallRoom.ID
		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
		}

		hallCenter, err := hallRoom.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		newDevice := device.NewDevice(deviceID, "smart_siren", "security")
		placement := device.NewPlacement(newDevice, roomID, *hallCenter)

		apartmentLayout.Placements[roomID][newDevice.Type] = placement
	}

	return nil
}
