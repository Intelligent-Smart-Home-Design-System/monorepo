package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type WaterLeakRule struct {
	track string
}

func NewWaterLeakRule() *WaterLeakRule {
	return &WaterLeakRule{
		track: "security",
	}
}

func (wl *WaterLeakRule) GetType() string {
	return "water_leak_sensor"
}

func (wl *WaterLeakRule) Apply(apartment *entities.Apartment) map[string]map[string]*entities.Placement {
	res := make(map[string]map[string]*entities.Placement)

	for _, room := range apartment.Rooms {
		for _, wetPoint := range room.WetPoints {
			ID := uuid.NewString() // в будущем все ID будут прописаны в конфигах
								   // и будут браться оттуда
			device := entities.NewDevice(ID, "water_leak_sensor", "security")
			placement := entities.NewPlacement(device, room.ID, wetPoint)
			res[room.ID] = make(map[string]*entities.Placement)
			res[room.ID][device.Type] = placement
		}
	}
	return res
}
