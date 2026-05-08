package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type WaterLeakSensorRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewWaterLeakRule(deviceConfig *configs.Devices) *WaterLeakSensorRule {
	return &WaterLeakSensorRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (wl *WaterLeakSensorRule) Type() string {
	return "water_leak_sensor"
}

func (wl *WaterLeakSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := wl.Type()

	configFilters := wl.deviceConfig.GetDeviceFilter(deviceType)
	typeFilters, err := filters.GetCertainFilter(deviceType, configFilters)
	if err != nil {
		return err
	}

	waterLeakSensorFilters := typeFilters.(*filters.WaterLeakSensorFilter)

	wetRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	// TODO: улучшить, когда появятся мокрые точки на плане

	for _, room := range wetRooms {
		roomID := room.ID

		roomCenter, err := room.GetCenter()
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(wl.Type(), wl.track, roomID, roomCenter, waterLeakSensorFilters)
	}

	return nil
}
