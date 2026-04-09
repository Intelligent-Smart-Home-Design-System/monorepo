package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/google/uuid"
)

type CameraRule struct {
	track string
}

func NewCameraRule() *CameraRule {
	return &CameraRule{
		track: "security",
	}
}

func (gl *CameraRule) GetType() string {
	return "camera"
}

func (gl *CameraRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	cameraRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range cameraRooms {
		roomID := room.ID

		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
		}

		cameraPoint, err := room.GetBestCameraPoint(apartmentStruct)
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		newDevice := device.NewDevice(deviceID, "camera", "security")
		placement := device.NewPlacement(newDevice, roomID, cameraPoint)

		apartmentLayout.Placements[roomID][newDevice.Type] = placement
	}

	return nil
}
