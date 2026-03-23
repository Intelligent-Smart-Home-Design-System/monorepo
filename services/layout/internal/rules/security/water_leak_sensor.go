package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type WaterLeakSensorRule struct {
	track string
}

func NewWaterLeakRule() *WaterLeakSensorRule {
	return &WaterLeakSensorRule{
		track: "security",
	}
}

func (wl *WaterLeakSensorRule) GetType() string {
	return "water_leak_sensor"
}

func (wl *WaterLeakSensorRule) Apply(apartment *entities.Apartment, apartmentLayout *entities.ApartmentLayout) error {
	wetRooms := apartment.GetRoomsByType("wet")

	for _, room := range wetRooms {
		roomID := room.ID

		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*entities.Placement)
		}

		roomCenter, err := room.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString() // в будущем все ID будут прописаны в конфигах и будут браться оттуда
		device := entities.NewDevice(deviceID, "water_leak_sensor", "security")
		placement := entities.NewPlacement(device, roomID, *roomCenter)

		apartmentLayout.Placements[roomID][device.Type] = placement
	}

	return nil
}
