package security

import (	
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/google/uuid"
)

type WaterLeakRule struct {
	ID string
	Track string
}

func NewWaterLeakRule(ID, track string) *WaterLeakRule {
	return &WaterLeakRule{ID: ID, Track: track} 
}

func (wl *WaterLeakRule) HasSuitableTrack(apartment *entities.Apartment) bool {
	for _, track := range apartment.Tracks {
		if track == wl.Track {
			return true
		}
	}
	return false
}

// Apply возвращает мапу, которая по roomID выдает расставленное устройство (объект структуры Placement)
// Через Apply устройство расставляется во всех нужных (по алгоритму) местах в каждой комнате
func (wl *WaterLeakRule) Apply(apartment *entities.Apartment) map[string]map[string]*entities.Placement {
	res := make(map[string]map[string]*entities.Placement)

	for _, room := range apartment.Rooms {
		for _, wetPoint := range room.WetPoints {
			ID := uuid.NewString() // в будущем все ID будут прописаны в конфигах,
								   // и все ID (устройств) будут браться оттуда
			device := entities.NewDevice(ID, "water_leak", "security")
			placement := entities.NewPlacement(device, room.ID, wetPoint)
			res[room.ID] = make(map[string]*entities.Placement)
			res[room.ID][device.ID] = placement
		}
	}
	return res
}
