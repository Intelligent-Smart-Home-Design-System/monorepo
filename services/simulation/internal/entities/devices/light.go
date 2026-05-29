package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Lamp реализует интерфейс entities.EntityWithProcess.
type Lamp struct {
	BaseDevice[LampData]
	TurnedOn bool `json:"turned_on"`
}

type LampData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

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
	l.TurnedOn = inData.TurnOn

	return LampData{
		Kind:   inData.Kind,
		TurnOn: l.TurnedOn,
	}
}

// SmartDimmer (Декоративный светильник — используется для акцентного освещения и сцен)
// реализует интерфейс entities.EntityWithProcess.
type SmartDimmer struct {
	BaseDevice[DimmerData]
	Brightness int `json:"brightness"` // 0-100
}

type DimmerData struct {
	Kind       string `json:"kind"`
	Brightness int    `json:"brightness"`
}

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
	brightness := inData.Brightness

	if brightness < 0 {
		brightness = 0
	}

	if brightness > 100 {
		brightness = 100
	}

	d.Brightness = brightness

	out := DimmerData{
		Kind:       inData.Kind,
		Brightness: d.Brightness,
	}

	return out
}
