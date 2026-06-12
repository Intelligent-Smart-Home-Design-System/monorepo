package configs

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type Tracks struct {
	Tracks map[string]*Track `json:"tracks"`
}

type Track struct {
	Name   string            `json:"name"`
	Levels map[string]*Level `json:"levels"`
}

type Level struct {
	Name            string                          `json:"name"`
	Description     string                          `json:"description"`
	PriceRange      PriceRange                      `json:"price_range"`
	Devices         []string                        `json:"devices"`
	DeviceRooms     map[string][]string             `json:"device_rooms"`
	MaxDeviceCounts map[string]int                  `json:"max_device_counts"`
	DeviceFilters  map[string]filters.DeviceFilter `json:"device_filters"`
}

type PriceRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

var globalTracksConfig *Tracks

func LoadTracksConfig(path string) error {
	var tracks Tracks

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &tracks)
	if err != nil {
		return err
	}

	for trackName, track := range tracks.Tracks {
		for levelNum, level := range track.Levels {
			typedFilters := make(map[string]filters.DeviceFilter)
			for deviceType, filter := range level.DeviceFilters {
				
				typedFilter, err := filters.GetCertainFilter(deviceType, filter)
				if err != nil {
					return err
				}
				if deviceType == "motion_sensor" {
				}
				typedFilters[deviceType] = typedFilter
			}

			tracks.Tracks[trackName].Levels[levelNum].DeviceFilters = typedFilters
		}
	}

	CreateGlobalTracksConfig(&tracks)

	return err
}

func CreateGlobalTracksConfig(config *Tracks) {
	globalTracksConfig = config
}

func GetGlobalTracksConfig() *Tracks {
	return globalTracksConfig
}

func (t *Tracks) GetDeviceFilter(trackName, levelNum, deviceType string) (filters.DeviceFilter, error) {
	track, ok := t.Tracks[trackName]
	if !ok {
		return nil, fmt.Errorf("Failed to load info about track %s", trackName)
	}

	level, ok := track.Levels[levelNum]
	if !ok {
		return nil, fmt.Errorf("Failed to load info about level %s in track %s", levelNum, trackName)
	}

	filter, _ := level.DeviceFilters[deviceType]
	return filter, nil
}
