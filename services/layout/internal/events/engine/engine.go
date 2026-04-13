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

func NewEngine(st *storage.Storage, tracksConfig *configs.Tracks, deviceConfig *configs.Devices) *Engine {
	return &Engine{
		storage:       st,
		tracksConfig:  tracksConfig,
		devicesConfig: deviceConfig,
	}
}

// PlaceDevices расставляет устройства по выбранному уровню в каждом треке
func (e *Engine) PlaceDevices(apartmentStruct *apartment.Apartment, selectedLevels map[string]string) (*apartment.ApartmentLayout, error) {
	if apartmentStruct == nil {
		return nil, fmt.Errorf("nil apartment")
	}

	if apartmentStruct.Rooms == nil {
		return nil, fmt.Errorf("nil rooms")
	}

	res := apartment.NewApartmentResult()

	for track, level := range selectedLevels {
		trackConfig := e.tracksConfig.Tracks[track]
		levelInfo, _ := trackConfig.Levels[level]

		for _, device := range levelInfo.Devices {
			rule, ok := e.storage.Rules[device]
			if !ok {
				return nil, fmt.Errorf("failed to get rule for device %s", device)
			}

			deviceRooms, ok := levelInfo.DeviceRooms[device]
			if !ok {
				return nil, fmt.Errorf("no info about rooms for device %s", device)
			}

			err := rule.Apply(apartmentStruct, deviceRooms, res)
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}

func (e *Engine) CalculateLayoutPrice(apartmentLayout *apartment.ApartmentLayout) *PriceInfo {
	priceInfo := &PriceInfo{}

	for _, roomPlacement := range apartmentLayout.Placements {
		for deviceType := range roomPlacement {
			device := e.devicesConfig.Devices[deviceType]

			priceInfo.MinPrice += device.Price.Min
			priceInfo.MaxPrice += device.Price.Max
		}
	}

	return priceInfo
}
