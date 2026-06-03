package configs

import (
	"encoding/json"
	"os"
)

var globalDevicesConfig *Devices

type Devices struct {
	Devices map[string]Device `json:"device_types"`
}

type Device struct {
	Name    string                 `json:"name"`
	Price   PriceRange             `json:"price_range"`
	Tracks  []string               `json:"tracks"`
}

func LoadDevicesConfig(path string) error {
	var devices Devices

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &devices)
	if err != nil {
		return err
	}

	CreateGlobalDevicesConfig(&devices)

	return err
}

func CreateGlobalDevicesConfig(config *Devices) {
	globalDevicesConfig = config
}

func GetGlobalDevicesConfig() *Devices {
	return globalDevicesConfig
}

// func (d *Devices) GetDeviceFilter(deviceType string) filters.DeviceFilter {
// 	if d.Filters != nil {
// 		return d.Filters[deviceType]
// 	}

// 	return nil
// }
