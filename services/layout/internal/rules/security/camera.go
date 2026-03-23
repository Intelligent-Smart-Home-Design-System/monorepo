package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
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

func (gl *CameraRule) Apply(apartment *entities.Apartment, apartmentLayout *entities.ApartmentLayout) error {
	cameraRooms := []string{"living", "hall"}

	for _, roomType := range cameraRooms {
		rooms := apartment.GetRoomsByType(roomType)

		for _, room := range rooms {
			roomID := room.ID

			_, ok := apartmentLayout.Placements[roomID]
			if !ok {
				apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
			}

			cameraPoint, err := room.GetBestCameraPoint(apartment)
			if err != nil {
				return err
			}

			deviceID := uuid.NewString()
			device := entities.NewDevice(deviceID, "camera", "security")
			placement := entities.NewPlacement(device, roomID, *cameraPoint)

			apartmentLayout.Placements[roomID][device.Type] = placement
		}
	}

	return nil
}
