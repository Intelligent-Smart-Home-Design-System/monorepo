package filters

import (
	"fmt"
	"reflect"
	"strings"
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
	Angle      string `json:"angle,omitempty"`
	Resolution string `json:"resolution,omitempty"`
}

type DoorSensorFilter struct{}

type WindowSensorFilter struct{}

type MotionSensorFilter struct {
	Angle int `json:"angle,omitempty"`
	Range int `json:"range,omitempty"`
}

type CameraFilter struct {
	Angle       int    `json:"angle,omitempty"`
	Range       int    `json:"range,omitempty"`
	NightVision bool   `json:"night_vision,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

type SmartSirenFilter struct {
	VolumeDB int `json:"volume_db,omitempty"`
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

	err := MapToFilterStruct(filters, filter)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

// MapToFilterStruct - это вспомогательная функция к GetCertainFilter
func MapToFilterStruct(filters map[string]interface{}, filterResult interface{}) error {
	v := reflect.ValueOf(filterResult)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("failed to convert map to certain filter struct")
	}

	v = v.Elem()
	t := v.Type()

	tagToField := make(map[string]reflect.Value)
	for i := range v.NumField() {
		field := v.Field(i)
		filedType := t.Field(i)

		tag := filedType.Tag.Get("json")
		tag = strings.Split(tag, ",")[0]
		tagToField[tag] = field
	}

	for tag, value := range filters {
		field, ok := tagToField[tag]
		if ok {
			v = reflect.ValueOf(value)
			if v.Type().ConvertibleTo(field.Type()) {
				field.Set(v.Convert(field.Type()))
			} else {
				_ = fmt.Errorf("failed to set field")
			}
		}
	}

	return nil
}
