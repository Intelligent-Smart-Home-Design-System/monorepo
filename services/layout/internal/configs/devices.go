package configs

import (
	"encoding/json"
	"os"
)

type Devices struct {
	Devices map[string]Device `json:"device_types"`
}

type Device struct {
	Name    string                 `json:"name"`
	Price   PriceRange             `json:"price_range"`
	Tracks  []string               `json:"tracks"`
	Filters map[string]interface{} `json:"filters"`
}

func LoadDevicesConfig(path string) (*Devices, error) {
	var devices Devices

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &devices)
	if err != nil {
		return nil, err
	}

	return &devices, err
}

func (d *Devices) GetDeviceFilter(deviceType string) map[string]interface{} {
	device, ok := d.Devices[deviceType]
	if ok {
		return device.Filters
	}

	return nil
}
