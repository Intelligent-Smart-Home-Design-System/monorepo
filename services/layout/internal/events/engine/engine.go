package engine

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
)

type Engine struct {
	storage *storage.Storage
}

func NewEngine(st *storage.Storage) *Engine {
	return &Engine{
		storage: st,
	}
}

// PlaceDevices расставляет устройства по выбранному уровню в каждом треке
func (e *Engine) PlaceDevices(ap *apartment.Apartment, selectedLevels map[string]string) (*apartment.Layout, error) {
	if ap == nil {
		return nil, fmt.Errorf("nil apartment")
	}

	if ap.Rooms == nil {
		return nil, fmt.Errorf("nil rooms")
	}

	zonedAp := apartment.Build(ap)

	tracksConfig := configs.GetGlobalTracksConfig()
	res := apartment.NewApartmentResult()

	for trackName, levelNum := range selectedLevels {
		trackConfig := tracksConfig.Tracks[trackName]
		levelInfo, _ := trackConfig.Levels[levelNum]

		for _, device := range levelInfo.Devices {
			rule, ok := e.storage.Rules[device]
			if !ok {
				return nil, fmt.Errorf("failed to get rule for device %s", device)
			}

			deviceRooms, ok := levelInfo.DeviceRooms[device]
			if !ok {
				return nil, fmt.Errorf("no info about rooms for device %s", device)
			}

			maxCount, ok := levelInfo.MaxDeviceCounts[device]
			if !ok {
				return nil, fmt.Errorf("no info about max count for device %s", device)
			}

			err := rule.Apply(zonedAp, levelNum, deviceRooms, maxCount, res)
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}

func (e *Engine) CalculateLayoutPrice(apartmentLayout *apartment.Layout) *PriceInfo {
	devicesConfig := configs.GetGlobalDevicesConfig()

	priceInfo := &PriceInfo{}
	for _, roomPlacements := range apartmentLayout.Placements {
		for _, placement := range roomPlacements {
			device := devicesConfig.Devices[placement.Device.Type]

			priceInfo.MinPrice += device.Price.Min
			priceInfo.MaxPrice += device.Price.Max
		}
	}

	return priceInfo
}
