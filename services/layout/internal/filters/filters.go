package filters

import (
	"encoding/json"
)

type DeviceFilter interface{}

// GetCertainFilter конвертирует словарь интерфейсов в структуру определенного устройства
func GetCertainFilter(deviceType string, filters interface{}) (DeviceFilter, error) {
	var filter DeviceFilter

	switch deviceType {

	// Security-устройства
	case "water_leak_sensor":
		filter = &WaterLeakSensorFilter{}
	case "gas_leak_sensor":
		filter = &GasLeakSensorFilter{}
	case "smart_lock":
		filter = &SmartLockFilter{}
	case "smart_doorbell":
		filter = &SmartDoorbellFilter{}
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

	// Climate-устройства
	case "air_conditioner":
		filter = &AirConditionerFilter{}
	case "temperature_sensor":
		filter = &TemperatureSensorFilter{}
	case "smart_radiator_actuator":
		filter = &SmartRadiatorActuatorFilter{}
	case "humidity_sensor":
		filter = &HumiditySensorFilter{}
	case "smart_humidifier":
		filter = &SmartHumidifierFilter{}
	case "co2_sensor":
		filter = &CO2SensorFilter{}
	case "air_purifier":
		filter = &AirPurifierFilter{}
	case "smart_floor_thermostat":
		filter = &SmartFloorThermostatFilter{}
	case "floor_temperature_sensor":
		filter = &FloorTemperatureSensorFilter{}

		// Household-устройства
	case "robot_vacuum":
		filter = &RobotVacuumFilter{}

	// Media-устройства
	case "smart_tv":
		filter = &SmartTVFilter{}
	case "smart_speaker":
		filter = &SmartSpeaker{}
	case "subwoofer":
		filter = &Subwoofer{}
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
