package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type SmartSirenRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewSmartSirenRule(deviceConfig *configs.Devices) *SmartSirenRule {
	return &SmartSirenRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (ss *SmartSirenRule) Type() string {
	return "smart_siren"
}

func (ss *SmartSirenRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := ss.Type()

	configFilters := ss.deviceConfig.GetDeviceFilter(deviceType)
	typeFilters, err := filters.GetCertainFilter(deviceType, configFilters)
	if err != nil {
		return err
	}

	smartSirenFilters := typeFilters.(*filters.SmartSirenFilter)

	hallRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, hallRoom := range hallRooms {
		roomID := hallRoom.ID

		hallCenter, err := hallRoom.GetCenter()
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(ss.Type(), ss.track, roomID, hallCenter, smartSirenFilters)
	}

	return nil
}
