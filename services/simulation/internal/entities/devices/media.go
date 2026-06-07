package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Мультимедия

// TV — телевизор (вкл/выкл).
type TV struct {
	BaseDevice[TVData]
	TurnOn bool `json:"turn_on"`
}

type TVData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewTV(data []byte, engineAPI engine.EnginePort) (*TV, error) {
	var tv TV
	if err := json.Unmarshal(data, &tv); err != nil {
		return nil, err
	}

	tv.enginePort = engineAPI
	tv.inStore = *simgo.NewStore[TVData](engineAPI.GetSimulation())
	tv.handler = tv.HandleEvent

	return &tv, nil
}

func (tv *TV) HandleInDTO(dto []byte) error {
	input := TVData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	tv.Put(input)

	return nil
}

func (tv *TV) HandleEvent(inData TVData) TVData {
	tv.TurnOn = inData.TurnOn

	return TVData{
		Kind:   inData.Kind,
		TurnOn: tv.TurnOn,
	}
}

// Subwoofer — сабвуфер (вкл/выкл).
type Subwoofer struct {
	BaseDevice[SubwooferData]
	TurnOn bool `json:"turn_on"`
}

type SubwooferData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSubwoofer(data []byte, engineAPI engine.EnginePort) (*Subwoofer, error) {
	var s Subwoofer
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	s.enginePort = engineAPI
	s.inStore = *simgo.NewStore[SubwooferData](engineAPI.GetSimulation())
	s.handler = s.HandleEvent

	return &s, nil
}

func (s *Subwoofer) HandleInDTO(dto []byte) error {
	input := SubwooferData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)

	return nil
}

func (s *Subwoofer) HandleEvent(inData SubwooferData) SubwooferData {
	s.TurnOn = inData.TurnOn

	return SubwooferData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}
}
