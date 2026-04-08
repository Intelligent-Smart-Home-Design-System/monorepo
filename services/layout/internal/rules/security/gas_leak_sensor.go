package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type GasLeakSensorRule struct {
	track string
}

func NewGasLeakRule() *GasLeakSensorRule {
	return &GasLeakSensorRule{
		track: "security",
	}
}

func (gl *GasLeakSensorRule) GetType() string {
	return "gas_leak_sensor"
}

func (gl *GasLeakSensorRule) Apply(apartment *entities.Apartment, deviceRooms []string, apartmentLayout *entities.ApartmentLayout) error {
	kitchens, err := apartment.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, kitchen := range kitchens {
		kitchenID := kitchen.ID

		_, ok := apartmentLayout.Placements[kitchenID]
		if !ok {
			apartmentLayout.Placements[kitchenID] = make(map[string]*entities.Placement)
		}

		kitchenCenter, err := kitchen.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		device := entities.NewDevice(deviceID, "gas_leak_sensor", "security")
		placement := entities.NewPlacement(device, kitchenID, *kitchenCenter)

		apartmentLayout.Placements[kitchenID][device.Type] = placement
	}

	return nil
}
