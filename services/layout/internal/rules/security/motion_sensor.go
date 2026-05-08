package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type MotionSensorRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewMotionSensorRule(deviceConfig *configs.Devices) *MotionSensorRule {
	return &MotionSensorRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (ms *MotionSensorRule) Type() string {
	return "motion_sensor"
}

func (ms *MotionSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := ms.Type()

	configFilters := ms.deviceConfig.GetDeviceFilter(deviceType)
	typeFilters, err := filters.GetCertainFilter(deviceType, configFilters)
	if err != nil {
		return err
	}

	motionSensorFilters := typeFilters.(*filters.MotionSensorFilter)

	motionRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range motionRooms {
		roomID := room.ID

		roomCenter, err := room.GetCenter()
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(deviceType, ms.track, roomID, roomCenter, motionSensorFilters)
	}

	return nil
}
