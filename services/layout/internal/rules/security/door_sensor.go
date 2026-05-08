package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type DoorSensorRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewDoorSensorRule(deviceConfig *configs.Devices) *DoorSensorRule {
	return &DoorSensorRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (ds *DoorSensorRule) Type() string {
	return "door_sensor"
}

func (ds *DoorSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := ds.Type()

	configFilters := ds.deviceConfig.GetDeviceFilter(deviceType)
	if configFilters == nil {
		configFilters = &filters.DoorSensorFilter{}
	}
	doorSensorFilters := configFilters.(*filters.DoorSensorFilter)

	frontDoor := apartmentStruct.GetFrontDoor()
	if frontDoor == nil {
		return fmt.Errorf("no front door")
	}

	roomID := frontDoor.Rooms[0]

	doorCenter := apartment.GetObjectCenter(frontDoor.Points)
	layout.AddDeviceToLayout(deviceType, ds.track, roomID, &doorCenter, doorSensorFilters)

	return nil
}
