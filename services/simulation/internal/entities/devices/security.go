package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Безопасность

// Siren — исполнительное устройство (умная сирена).
// Активируется по сигналу тревоги и генерирует звуковое/световое оповещение.
type Siren struct {
	BaseDevice[SirenData]
	TurnOn bool `json:"turn_on"`
}

type SirenData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSiren(data []byte, engineAPI engine.EnginePort) (*Siren, error) {
	var s Siren

	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	s.enginePort = engineAPI
	s.inStore = *simgo.NewStore[SirenData](engineAPI.GetSimulation())
	s.handler = s.HandleEvent

	return &s, nil
}

func (s *Siren) HandleInDTO(dto []byte) error {
	input := SirenData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)
	return nil
}

// HandleEvent — бизнес-логика сирены.
func (s *Siren) HandleEvent(inData SirenData) SirenData {
	s.TurnOn = inData.TurnOn

	return SirenData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}
}
