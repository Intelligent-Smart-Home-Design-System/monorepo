package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
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

func (gl *GasLeakSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	kitchens, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, kitchen := range kitchens {
		kitchenID := kitchen.ID

		_, ok := apartmentLayout.Placements[kitchenID]
		if !ok {
			apartmentLayout.Placements[kitchenID] = make(map[string]*device.Placement)
		}

		kitchenCenter, err := kitchen.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		newDevice := device.NewDevice(deviceID, "gas_leak_sensor", "security")
		placement := device.NewPlacement(newDevice, kitchenID, kitchenCenter)

		apartmentLayout.Placements[kitchenID][newDevice.Type] = placement
	}

	return nil
}
