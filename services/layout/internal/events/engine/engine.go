package engine

import (
	"encoding/json"
	"fmt"
	"os"

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
	res := apartment.NewLayout()

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

// MakeScenarioDependencies возвращает зависимости между устройства внутри итоговой расстановки
func (e *Engine) MakeScenarioDependencies(layout *apartment.Layout) (map[string][]string, error) {
	result := make(map[string][]string)

	var dependencies struct {
		Triggers map[string]TriggerInfo
	}
	
	path := "../../../../simulation/configs/dependencies.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &dependencies); err != nil {
		return nil, err
	}

	roomAndTypeToDeviceIDs := make(map[string]map[string][]string)
	smartSirens := make([]string, 0)

	for roomID, placements := range layout.Placements {
		for _, placement := range placements {
			if placement.Device == nil {
				continue
			}

			deviceType := placement.Device.Type
			if deviceType == "smart_siren" {
				smartSirens = append(smartSirens, placement.Device.ID)
			}

			if roomAndTypeToDeviceIDs[roomID] == nil {
				roomAndTypeToDeviceIDs[roomID] = make(map[string][]string)
			}


			roomAndTypeToDeviceIDs[roomID][deviceType] = append(
				roomAndTypeToDeviceIDs[roomID][deviceType], 
				placement.Device.ID,
			)
		}
	}

	for triggerType, info := range dependencies.Triggers {
		for _, typeToDeviceIDs := range roomAndTypeToDeviceIDs {
			triggersIDs, ok := typeToDeviceIDs[triggerType]
			if !ok {
				continue
			}

			executorsIDs := make([]string, 0)
			isUsedSirens := false
			for _, executorType := range info.Triggers {
				if !isUsedSirens && executorType == "smart_siren" {
					executorsIDs = append(executorsIDs, smartSirens...)
					isUsedSirens = true
				} else {
					executorsInRoom, ok := typeToDeviceIDs[executorType]
					if ok {
						executorsIDs = append(executorsIDs, executorsInRoom...)
					}
				}
			}

			if len(executorsIDs) == 0 {
				continue
			}

			for _, triggerID := range triggersIDs {
				result[triggerID] = executorsIDs
			}
		}
	}

	return result, nil
}

func (e *Engine) CalculateLayoutPrice(layout *apartment.Layout) *PriceInfo {
	devicesConfig := configs.GetGlobalDevicesConfig()

	priceInfo := &PriceInfo{}
	for _, roomPlacements := range layout.Placements {
		for _, placement := range roomPlacements {
			device := devicesConfig.Devices[placement.Device.Type]

			priceInfo.MinPrice += device.Price.Min
			priceInfo.MaxPrice += device.Price.Max
		}
	}

	return priceInfo
}
