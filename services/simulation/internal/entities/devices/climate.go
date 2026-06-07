package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Климат

// AirConditioner — кондиционер (вкл/выкл + температура).
type AirConditioner struct {
	BaseDevice[AirConditionerData]
	TurnOn      bool    `json:"turn_on"`
	Temperature float64 `json:"temperature"`
}

// AirConditionerData — данные для кондиционера, используемые в обработчике событий.
type AirConditionerData struct {
	Kind        string   `json:"kind"`
	TurnOn      *bool    `json:"turn_on"`
	Temperature *float64 `json:"temperature"`
}

// NewAirConditioner - конструктор для создания нового кондиционера из JSON-данных и API движка.
func NewAirConditioner(data []byte, engineAPI engine.EnginePort) (*AirConditioner, error) {
	var ac AirConditioner
	if err := json.Unmarshal(data, &ac); err != nil {
		return nil, err
	}

	ac.enginePort = engineAPI
	ac.inStore = *simgo.NewStore[AirConditionerData](engineAPI.GetSimulation())
	ac.handler = ac.HandleEvent

	return &ac, nil
}

// HandleInDTO - метод для обработки входящих данных в формате JSON, обновляющий состояние кондиционера.
func (ac *AirConditioner) HandleInDTO(dto []byte) error {
	input := AirConditionerData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}
	ac.Put(input)

	return nil
}

// HandleEvent - метод для обработки событий, обновляющий состояние кондиционера на основе входных данных и возвращающий текущее состояние.
func (ac *AirConditioner) HandleEvent(inData AirConditionerData) AirConditionerData {
	if inData.TurnOn != nil {
		ac.TurnOn = *inData.TurnOn
	}

	if inData.Temperature != nil {
		ac.Temperature = *inData.Temperature
	}

	return AirConditionerData{
		Kind:        inData.Kind,
		TurnOn:      &ac.TurnOn,
		Temperature: &ac.Temperature,
	}
}

// Thermostat — терморегулятор (вкл/выкл + процент нагрева 0-100).
type Thermostat struct {
	BaseDevice[ThermostatData]
	TurnOn      bool `json:"turn_on"`
	Temperature int  `json:"temperature"` // 0-100
}

type ThermostatData struct {
	Kind        string `json:"kind"`
	TurnOn      *bool  `json:"turn_on"`
	Temperature *int   `json:"temperature"`
}

// NewThermostat - конструктор для создания нового термостата из JSON-данных и API движка.
func NewThermostat(data []byte, engineAPI engine.EnginePort) (*Thermostat, error) {
	var t Thermostat
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	t.enginePort = engineAPI
	t.inStore = *simgo.NewStore[ThermostatData](engineAPI.GetSimulation())
	t.handler = t.HandleEvent

	return &t, nil
}

// HandleInDTO - метод для обработки входящих данных в формате JSON, обновляющий состояние термостата.
func (t *Thermostat) HandleInDTO(dto []byte) error {
	input := ThermostatData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	t.Put(input)

	return nil
}

// HandleEvent - метод для обработки событий, обновляющий состояние термостата на основе входных данных и возвращающий текущее состояние.
func (t *Thermostat) HandleEvent(inData ThermostatData) ThermostatData {
	if inData.TurnOn != nil {
		t.TurnOn = *inData.TurnOn
	}

	if inData.Temperature != nil {
		temp := *inData.Temperature
		if temp < 0 {
			temp = 0
		}
		if temp > 100 {
			temp = 100
		}
		t.Temperature = temp
	}

	return ThermostatData{
		Kind:        inData.Kind,
		TurnOn:      &t.TurnOn,
		Temperature: &t.Temperature,
	}
}
