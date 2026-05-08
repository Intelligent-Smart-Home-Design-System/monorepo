package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type SmartDoorBellRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewSmartDoorBellRule(deviceConfig *configs.Devices) *SmartDoorBellRule {
	return &SmartDoorBellRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (sd *SmartDoorBellRule) Type() string {
	return "smart_doorbell"
}

func (sd *SmartDoorBellRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := sd.Type()

	configFilters := sd.deviceConfig.GetDeviceFilter(deviceType)
	typeFilters, err := filters.GetCertainFilter(deviceType, configFilters)
	if err != nil {
		return err
	}

	smartDoorBellFilters := typeFilters.(*filters.SmartDoorBellFilter)

	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door in apartment")
	}

	roomID := frontDoor.Rooms[0]
	layout.AddDeviceToLayout(sd.Type(), sd.track, roomID, &frontDoor.Points[0], smartDoorBellFilters)

	return nil
}
