package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type SmartLockRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewSmartLockRule(deviceConfig *configs.Devices) *SmartLockRule {
	return &SmartLockRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (sl *SmartLockRule) Type() string {
	return "smart_lock"
}

func (sl *SmartLockRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := sl.Type()

	configFilters := sl.deviceConfig.GetDeviceFilter(deviceType)
	typeFilters, err := filters.GetCertainFilter(deviceType, configFilters)
	if err != nil {
		return err
	}

	smartLockFilters := typeFilters.(*filters.SmartLockFilter)

	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door in apartment")
	}

	roomID := frontDoor.Rooms[0]

	doorCenter := apartment.GetObjectCenter(frontDoor.Points)
	layout.AddDeviceToLayout(sl.Type(), sl.track, roomID, &doorCenter, smartLockFilters)

	return nil
}
