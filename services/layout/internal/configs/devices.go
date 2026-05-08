package configs

import (
	"encoding/json"
	"os"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
)

type Devices struct {
	Devices map[string]Device `json:"device_types"`
	Filters map[string]filters.DeviceFilter
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

	devices.Filters = make(map[string]filters.DeviceFilter)
	for deviceType, device := range devices.Devices {
		if device.Filters != nil {
			filter, err := filters.GetCertainFilter(deviceType, device.Filters)
			if err != nil {
				return nil, err
			}
			
			devices.Filters[deviceType] = filter
		}
	}

	return &devices, err
}

func (d *Devices) GetDeviceFilter(deviceType string) filters.DeviceFilter {
	if d.Filters != nil {
		return d.Filters[deviceType]
	}

	return nil
}
