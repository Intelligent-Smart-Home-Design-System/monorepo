package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
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

// Camera — камера наблюдения с радиусом обзора.
type Camera struct {
	BaseDevice[CameraData]
	TurnOn bool    `json:"turn_on"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

type CameraData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewCamera(data []byte, engineAPI engine.EnginePort) (*Camera, error) {
	var c Camera
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	c.enginePort = engineAPI
	c.inStore = *simgo.NewStore[CameraData](engineAPI.GetSimulation())
	c.handler = c.HandleEvent

	return &c, nil
}

func (c *Camera) GetPosition() (float64, float64) {
	return c.X, c.Y
}

func (c *Camera) HandleInDTO(dto []byte) error {
	var move struct {
		Kind string `json:"kind"`
		To   struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"to"`
	}
	if err := json.Unmarshal(dto, &move); err != nil {
		return err
	}

	c.Put(CameraData{
		Kind:   move.Kind,
		TurnOn: field.IsInRadius(c.X, c.Y, move.To.X, move.To.Y, c.Radius),
	})

	return nil
}

func (c *Camera) HandleEvent(inData CameraData) CameraData {
	c.TurnOn = inData.TurnOn

	return CameraData{
		Kind:   inData.Kind,
		TurnOn: c.TurnOn,
	}
}

func (c *Camera) GetObservedKinds() []string {
	return []string{"human:move", "device:move"}
}
