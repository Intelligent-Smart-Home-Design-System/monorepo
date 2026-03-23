package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type SecurityAlarmRule struct {
	track string
}

func NewSecurityAlarmRule() *SecurityAlarmRule {
	return &SecurityAlarmRule{
		track: "security",
	}
}

func (gl *SecurityAlarmRule) GetType() string {
	return "security_alarm"
}

func (gl *SecurityAlarmRule) Apply(apartment *entities.Apartment, apartmentLayout *entities.ApartmentLayout) error {
	hallRoom := apartment.GetRoomsByType("hall")

	roomID := hallRoom[0].ID
	_, ok := apartmentLayout.Placements[roomID]
	if !ok {
		apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
	}

	hallCenter, err := hallRoom[0].GetCenter()
	if err != nil {
		return err
	}

	deviceID := uuid.NewString()
	device := entities.NewDevice(deviceID, "security_alarm", "security")
	placement := entities.NewPlacement(device, roomID, *hallCenter)

	apartmentLayout.Placements[roomID][device.Type] = placement

	return nil
}
