package filters

type DeviceFilter interface{}

type WaterLeakSensorFilter struct {
	Sensitivity int `json:"sensitivity,omitempty"`
	IsWireless bool `json:"is_wireless,omitempty"`
	AlertVolume int `json:"alert_volume,omitempty"`
}

type CameraFilter struct {
	Angle *float64 `json:"angle,omitempty"`
	VisibilityMetersRange *float64 `json:"visibility_range,omitempty"`
}

// TODO: добавить фильтры для остальных устройств
