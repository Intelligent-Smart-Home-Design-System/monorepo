package configs

import (
	"encoding/json"
	"os"
)

type Tracks struct {
	Tracks map[string]Track `json:"tracks"`
}

type Track struct {
	Name string `json:"name"`
	Levels map[string]Level `json:"levels"`
}

type Level struct {
	Name string `json:"name"`
	Description string `json:"description"`
	PriceRange PriceRange `json:"price_range"`
	Devices []string `json:"devices"`
}

type PriceRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func LoadTracksConfig(path string) (*Tracks, error) {
	var tracks Tracks

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &tracks)
	if err != nil {
		return nil, err
	}

	return &tracks, err
}
