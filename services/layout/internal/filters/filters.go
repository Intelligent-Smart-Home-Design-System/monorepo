package filters

import (
	"encoding/json"
)

type DeviceFilter interface{}

type WaterLeakSensorFilter struct{}

type GasLeakSensorFilter struct {
	GasType string `json:"gas_type,omitempty"`
}

type SmartLockFilter struct {
	UnlockMethods []string `json:"unlock_methods,omitempty"`
}

type SmartDoorBellFilter struct {
	Angle      float64 `json:"angle,omitempty"`
	Resolution string `json:"resolution,omitempty"`
}

type DoorSensorFilter struct{}

type WindowSensorFilter struct{}

type MotionSensorFilter struct {
	Angle float64 `json:"angle,omitempty"`
	Range float64 `json:"range,omitempty"`
}

type CameraFilter struct {
	Angle       float64    `json:"angle,omitempty"`
	Range       float64    `json:"range,omitempty"`
	NightVision bool   `json:"night_vision,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

type SmartSirenFilter struct {
	VolumeDB float64 `json:"volume_db,omitempty"`
}

// GetCertainFilter конвертирует словарь интерфейсов в структуру определенного устройства
func GetCertainFilter(deviceType string, filters map[string]interface{}) (DeviceFilter, error) {
	var filter DeviceFilter

	switch deviceType {
	case "water_leak_sensor":
		filter = &WaterLeakSensorFilter{}
	case "gas_leak_sensor":
		filter = &GasLeakSensorFilter{}
	case "smart_lock":
		filter = &SmartLockFilter{}
	case "smart_doorbell":
		filter = &SmartDoorBellFilter{}
	case "door_sensor":
		filter = &DoorSensorFilter{}
	case "window_sensor":
		filter = &WindowSensorFilter{}
	case "motion_sensor":
		filter = &MotionSensorFilter{}
	case "camera":
		filter = &CameraFilter{}
	case "smart_siren":
		filter = &SmartSirenFilter{}
	}

	if filter == nil {
		return filter, nil
	}

	data, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, filter)
	if err != nil {
		return nil, err
	}

	return filter, nil
}
