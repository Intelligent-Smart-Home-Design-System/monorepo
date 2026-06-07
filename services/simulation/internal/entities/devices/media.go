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

// TVData — данные для телевизора.
type TVData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewTV — конструктор телевизора.
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

// HandleInDTO - обработка входящих данных для телевизора.
func (tv *TV) HandleInDTO(dto []byte) error {
	input := TVData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	tv.Put(input)

	return nil
}

// HandleEvent - обработка события для телевизора.
func (tv *TV) HandleEvent(inData TVData) TVData {
	tv.TurnOn = inData.TurnOn

	return TVData{
		Kind:   inData.Kind,
		TurnOn: tv.TurnOn,
	}
}

// Subwoofer - сабвуфер (вкл/выкл).
type Subwoofer struct {
	BaseDevice[SubwooferData]
	TurnOn bool `json:"turn_on"`
}

// SubwooferData - данные для сабвуфера.
type SubwooferData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewSubwoofer - конструктор сабвуфера.
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

// HandleInDTO - обработка входящих данных для сабвуфера.
func (s *Subwoofer) HandleInDTO(dto []byte) error {
	input := SubwooferData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)

	return nil
}

// HandleEvent - обработка события для сабвуфера.
func (s *Subwoofer) HandleEvent(inData SubwooferData) SubwooferData {
	s.TurnOn = inData.TurnOn

	return SubwooferData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}
}
