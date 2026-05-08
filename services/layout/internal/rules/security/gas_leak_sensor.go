package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type GasLeakSensorRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewGasLeakRule(deviceConfig *configs.Devices) *GasLeakSensorRule {
	return &GasLeakSensorRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (gl *GasLeakSensorRule) Type() string {
	return "gas_leak_sensor"
}

func (gl *GasLeakSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := gl.Type()

	configFilters := gl.deviceConfig.GetDeviceFilter(deviceType)
	typeFilters, err := filters.GetCertainFilter(deviceType, configFilters)
	if err != nil {
		return err
	}

	gasLeakSensorFilters := typeFilters.(*filters.GasLeakSensorFilter)

	kitchens, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, kitchen := range kitchens {
		kitchenID := kitchen.ID

		kitchenCenter, err := kitchen.GetCenter()
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(deviceType, gl.track, kitchenID, kitchenCenter, gasLeakSensorFilters)
	}

	return nil
}
