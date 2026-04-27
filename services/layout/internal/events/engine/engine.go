package engine

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
)

type Engine struct {
	storage       *storage.Storage
	tracksConfig  *configs.Tracks
	devicesConfig *configs.Devices
}

func NewEngine(storage *storage.Storage, tracksConfig *configs.Tracks, deviceConfig *configs.Devices) *Engine {
	return &Engine{
		storage:       storage,
		tracksConfig:  tracksConfig,
		devicesConfig: deviceConfig,
	}
}

func (e *Engine) PlaceDevices(ap *apartment.Apartment, selectedLevels map[string]string) (*apartment.ApartmentResult, error) {
	if ap == nil {
		return nil, fmt.Errorf("nil apartment")
	}

	if ap.Rooms == nil {
		return nil, fmt.Errorf("nil rooms")
	}

	res := apartment.NewApartmentResult()

	for _, track := range ap.Tracks {
		trackConfig := e.tracksConfig.Tracks[track]
		level, ok := trackConfig.Levels[selectedLevels[track]]
		if !ok {
			level = trackConfig.Levels[BaseLevel] // по дефолту первый уровень во всех треках
		}

		for _, device := range level.Devices {
			rule, err := e.storage.GetRule(device)
			if err != nil {
				return nil, err
			}

			res.Placements = rule.Apply(ap)
		}
	}

	return res, nil
}

func (e *Engine) CalcLayoutPrice(apartmentResult *apartment.ApartmentResult) *PriceInfo {
	priceInfo := &PriceInfo{}

	for _, roomPlacement := range apartmentResult.Placements {
		for deviceType := range roomPlacement {
			device := e.devicesConfig.Devices[deviceType]

			priceInfo.MinPrice += device.Price.Min
			priceInfo.MaxPrice += device.Price.Max
		}
	}

	return priceInfo
}
