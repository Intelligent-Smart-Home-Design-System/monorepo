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

func (ss *SmartSirenRule) Type() string {
	return "smart_siren"
}

func (ss *SmartSirenRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	hallRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, hallRoom := range hallRooms {
		roomID := hallRoom.ID
		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make([]*device.Placement, 0)
		}

		hallCenter, err := hallRoom.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		newDevice := device.NewDevice(deviceID, ss.Type(), ss.track)
		placement := device.NewPlacement(newDevice, hallCenter, nil)

		apartmentLayout.Placements[roomID] = append(apartmentLayout.Placements[roomID], placement)
	}

	return nil
}
