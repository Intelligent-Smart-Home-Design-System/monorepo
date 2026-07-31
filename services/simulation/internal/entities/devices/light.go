package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Lamp реализует интерфейс entities.EntityWithProcess.
type Lamp struct {
	BaseDevice[LampData]
	TurnOn bool `json:"turn_on"`
}

// LampData - входные и выходные данные для лампы.
type LampData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewLamp - конструктор для создания новой лампы из JSON-данных.
func NewLamp(data []byte, engineAPI engine.EnginePort) (*Lamp, error) {
	var lamp Lamp
	if err := json.Unmarshal(data, &lamp); err != nil {
		return nil, err
	}

	lamp.enginePort = engineAPI
	lamp.inStore = *simgo.NewStore[LampData](engineAPI.GetSimulation())
	lamp.handler = lamp.HandleEvent

	return &lamp, nil
}

// HandleInDTO - метод для обработки входных данных в формате JSON и сохранения их во внутреннем хранилище.
func (l *Lamp) HandleInDTO(dto []byte) error {
	input := LampData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	l.Put(input)

	return nil
}

// HandleEvent реализует бизнес-логику устройства.
// Возвращает обработанные данные.
func (l *Lamp) HandleEvent(inData LampData) LampData {
	l.TurnOn = inData.TurnOn

	return LampData{
		Kind:   inData.Kind,
		TurnOn: l.TurnOn,
	}
}

// SmartLamp (управление светом — используется для акцентного освещения и сцен)
type SmartLamp struct {
	BaseDevice[SmartLampData]
	TurnOn   bool `json:"turn_on"`
	Percents int  `json:"percents"`
}

// SmartLampData - входные и выходные данные для SmartLamp.
type SmartLampData struct {
	Kind     string `json:"kind"`
	TurnOn   *bool  `json:"turn_on,omitempty"`
	Percents *int   `json:"percents,omitempty"`
}

// NewSmartLamp - конструктор для создания новой SmartLamp из JSON-данных.
func NewSmartLamp(data []byte, engineAPI engine.EnginePort) (*SmartLamp, error) {
	var lamp SmartLamp
	if err := json.Unmarshal(data, &lamp); err != nil {
		return nil, err
	}

	lamp.enginePort = engineAPI
	lamp.inStore = *simgo.NewStore[SmartLampData](engineAPI.GetSimulation())
	lamp.handler = lamp.HandleEvent

	return &lamp, nil
}

// HandleInDTO - метод для обработки входных данных в формате JSON и сохранения их во внутреннем хранилище.
func (s *SmartLamp) HandleInDTO(dto []byte) error {
	input := SmartLampData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)

	return nil
}

// HandleEvent реализует бизнес-логику устройства.
// Возвращает обработанные данные.
func (s *SmartLamp) HandleEvent(inData SmartLampData) SmartLampData {
	if inData.TurnOn != nil {
		s.TurnOn = *inData.TurnOn
	}
	if inData.Percents != nil {
		s.Percents = clampPercent(*inData.Percents)
	}

	return SmartLampData{
		Kind:     inData.Kind,
		TurnOn:   &s.TurnOn,
		Percents: &s.Percents,
	}
}

// SmartDimmer (управление светом - используется для акцентного освещения и сцен)
// реализует интерфейс entities.EntityWithProcess.
type SmartDimmer struct {
	BaseDevice[DimmerData]
	TurnOn   bool `json:"turn_on"`
	Percents int  `json:"percents"` // 0-100
}

// DimmerData - входные и выходные данные для SmartDimmer.
type DimmerData struct {
	Kind     string `json:"kind"`
	TurnOn   *bool  `json:"turn_on,omitempty"`
	Percents *int   `json:"percents,omitempty"`
}

// NewSmartDimmer - конструктор для создания новой SmartDimmer из JSON-данных.
func NewSmartDimmer(data []byte, engineAPI engine.EnginePort) (*SmartDimmer, error) {
	var dimmer SmartDimmer

	if err := json.Unmarshal(data, &dimmer); err != nil {
		return nil, err
	}

	dimmer.enginePort = engineAPI
	dimmer.inStore = *simgo.NewStore[DimmerData](engineAPI.GetSimulation())
	dimmer.handler = dimmer.HandleEvent

	return &dimmer, nil
}

// HandleInDTO - метод для обработки входных данных в формате JSON и сохранения их во внутреннем хранилище.
func (d *SmartDimmer) HandleInDTO(dto []byte) error {
	input := DimmerData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	d.Put(input)

	return nil
}

// HandleEvent реализует бизнес-логику диммера.
func (d *SmartDimmer) HandleEvent(inData DimmerData) DimmerData {
	if inData.TurnOn != nil {
		d.TurnOn = *inData.TurnOn
	}
	if inData.Percents != nil {
		d.Percents = clampPercent(*inData.Percents)
	}

	out := DimmerData{
		Kind:     inData.Kind,
		TurnOn:   &d.TurnOn,
		Percents: &d.Percents,
	}

	return out
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
