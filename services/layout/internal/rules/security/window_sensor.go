package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type WindowSensorRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewWindowSensorRule(deviceConfig *configs.Devices) *WindowSensorRule {
	return &WindowSensorRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (ws *WindowSensorRule) Type() string {
	return "window_sensor"
}

func (ws *WindowSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := ws.Type()

	configFilters := ws.deviceConfig.GetDeviceFilter(deviceType)
	if configFilters == nil {
		configFilters = &filters.WindowSensorFilter{}
	}
	windowSensorFilters := configFilters.(*filters.WindowSensorFilter)

	for _, window := range apartmentStruct.Windows {
		if len(window.Rooms) > 1 {
			continue
		}

		roomID := window.Rooms[0]

		windowCenter := apartment.GetObjectCenter(window.Points)
		layout.AddDeviceToLayout(ws.Type(), ws.track, roomID, &windowCenter, windowSensorFilters)
	}

	return nil
}
