package filters

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type DeviceFilter interface{}

type WaterLeakSensorFilter struct {
	ProbeType        string  `json:"probe_type,omitempty"`
	AlarmVolumeDB    float64 `json:"alarm_volume_db,omitempty"`
	BatteryLifeYears float64 `json:"battery_life_years,omitempty"`
}

type GasLeakSensorFilter struct {
	GasTypes         []string `json:"gas_types,omitempty"`
	AlarmVolumeDB    float64  `json:"alarm_volume_db,omitempty"`
	BatteryLifeYears float64  `json:"battery_life_years,omitempty"`
}

type SmartLockFilter struct {
	UnlockMethods []string `json:"unlock_methods,omitempty"`
}

type SmartDoorBellFilter struct {
	Resolution  string  `json:"resolution,omitempty"`
	Angle       float64 `json:"angle,omitempty"`
	NightVision bool    `json:"night_vision,omitempty"`
	TwoWayAudio bool    `json:"two_way_audio,omitempty"`

	Direction *point.Point
}

type DoorSensorFilter struct{}

type WindowSensorFilter struct{}

type MotionSensorFilter struct {
	Angle float64 `json:"angle,omitempty"`
	Range float64 `json:"range,omitempty"`

	Direction *point.Point
}

type CameraFilter struct {
	Angle       float64 `json:"angle,omitempty"`
	Range       float64 `json:"range,omitempty"`
	NightVision bool    `json:"night_vision,omitempty"`
	Resolution  string  `json:"resolution,omitempty"`

	RecommendedRange float64
	Direction        *point.Point
}

type SmartSirenFilter struct {
	VolumeDB float64 `json:"volume_db,omitempty"`
}

type AirConditionerFilter struct {
	NoiseLevelDB       float64 `json:"noise_level_db,omitempty"`
	MaxNoiseLevelDB    float64 `json:"max_noise_level_db,omitempty"`
	CoolingPowerBTU    float64 `json:"cooling_power_btu,omitempty"`
	CoolingPowerWatts  float64 `json:"cooling_power_watts,omitempty"`
	IndoorUnitLengthMM float64 `json:"indoor_unit_length_mm,omitempty"`
	RecommendedAreaM2  float64 `json:"recommended_area_m2,omitempty"`
}

// GetCertainFilter конвертирует словарь интерфейсов в структуру определенного устройства
func GetCertainFilter(deviceType string, filters interface{}) (DeviceFilter, error) {
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
	case "air_conditioner":
		filter = &AirConditionerFilter{}
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
