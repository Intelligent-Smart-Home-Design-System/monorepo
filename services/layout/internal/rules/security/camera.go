package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type CameraRule struct {
	track string
	deviceConfig *configs.Devices
}

func NewCameraRule(deviceConfig *configs.Devices) *CameraRule {
	return &CameraRule{
		track: "security",
		deviceConfig: deviceConfig,
	}
}

func (c *CameraRule) Type() string {
	return "camera"
}

func (c *CameraRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	deviceType := c.Type()

	configFilters := c.deviceConfig.GetDeviceFilter(deviceType)
	typeFilters, err := filters.GetCertainFilter(deviceType, configFilters)
	if err != nil {
		return err
	}

	cameraFilters := typeFilters.(*filters.CameraFilter)

	cameraRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range cameraRooms {
		roomID := room.ID

		maxDistance := int(room.CalculateMaxDistance() * 1.5)
		cameraFilters.Range = maxDistance

		cameraPoint, err := room.GetBestCameraPoint(apartmentStruct, cameraFilters)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(deviceType, c.track, roomID, cameraPoint, cameraFilters)
	}

	return nil
}
