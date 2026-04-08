package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
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

func (wl *WaterLeakRule) Apply(ap *apartment.Apartment) map[string]map[string]*device.Placement {
	res := make(map[string]map[string]*device.Placement)

	for _, room := range ap.Rooms {
		for _, wetPoint := range room.WetPoints {
			ID := uuid.NewString() // в будущем все ID будут прописаны в конфигах
			// и будут браться оттуда
			dev := device.NewDevice(ID, wl.GetType(), wl.track)
			placement := device.NewPlacement(dev, room.ID, wetPoint)
			res[room.ID] = make(map[string]*device.Placement)
			res[room.ID][dev.Type] = placement
		}
	}
	return res
}
