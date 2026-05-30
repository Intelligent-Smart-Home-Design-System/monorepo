package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Переключатели / розетки

// Switcher - стандартный переключатель с функцией вкл/выкл
type Switcher struct {
	BaseDevice[SwitcherData]
	TurnedOn bool `json:"turn_on"`
}

type SwitcherData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSwitcher(data []byte, engineAPI engine.EnginePort) (*Switcher, error) {
	var Switcher Switcher
	if err := json.Unmarshal(data, &Switcher); err != nil {
		return nil, err
	}

	Switcher.enginePort = engineAPI
	Switcher.inStore = *simgo.NewStore[SwitcherData](engineAPI.GetSimulation())
	Switcher.handler = Switcher.HandleEvent

	return &Switcher, nil
}

func (l *Switcher) HandleInDTO(dto []byte) error {
	input := SwitcherData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	l.Put(input)

	return nil
}

func (l *Switcher) HandleEvent(inData SwitcherData) SwitcherData {
	l.TurnedOn = inData.TurnOn

	out := SwitcherData{
		Kind:   inData.Kind,
		TurnOn: l.TurnedOn,
	}

	return out
}
