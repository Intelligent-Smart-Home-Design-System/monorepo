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
	TurnOn bool `json:"turn_on"`
}

type SwitcherData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSwitcher(data []byte, engineAPI engine.EnginePort) (*Switcher, error) {
	var sw Switcher
	if err := json.Unmarshal(data, &sw); err != nil {
		return nil, err
	}

	sw.enginePort = engineAPI
	sw.inStore = *simgo.NewStore[SwitcherData](engineAPI.GetSimulation())
	sw.handler = sw.HandleEvent

	return &sw, nil
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
	l.TurnOn = inData.TurnOn

	out := SwitcherData{
		Kind:   inData.Kind,
		TurnOn: l.TurnOn,
	}

	return out
}
