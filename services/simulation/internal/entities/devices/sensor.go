package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Датчики

// SensorWithUpdate - сенсор-переключатель с функцией обновления последнего действия.
// Позволяет обновлять действие до истечения таймаута, старое действие игнорируется.
type SensorWithUpdate struct {
	BaseDevice[SensorWithUpdateData]
	TurnedOn bool    `json:"turn_on"`
	Timeout  float64 `json:"timeout"`
}

type SensorWithUpdateData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSensorWithUpdate(data []byte, engineAPI engine.EnginePort) (*SensorWithUpdate, error) {
	var switcher SensorWithUpdate
	if err := json.Unmarshal(data, &switcher); err != nil {
		return nil, err
	}

	switcher.enginePort = engineAPI
	switcher.inStore = *simgo.NewStore[SensorWithUpdateData](engineAPI.GetSimulation())
	return &switcher, nil
}

func (s *SensorWithUpdate) HandleInDTO(dto []byte) error {
	input := SensorWithUpdateData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)
	return nil
}

func (s *SensorWithUpdate) GetProcessFunc() func(process simgo.Process) {
	return s.Process
}

func (s *SensorWithUpdate) Process(process simgo.Process) {
	for {
		el := s.inStore.Get()
		process.Wait(el.Event)
		process.Wait(process.Timeout(s.Delay))

		inData := el.Item
		if !inData.TurnOn {
			continue
		}

		outData, err := s.HandleEvent(SensorWithUpdateData{TurnOn: true})
		if err != nil {
			slog.Warn("handler error", "id", s.ID, "err", err)
			continue
		}

		dataLamp, _ := json.Marshal(outData)
		s.HandleOutDTO(dataLamp)

		for s.TurnedOn {
			timeoutEv := process.Timeout(s.Timeout)
			el2 := s.inStore.Get()

			process.Wait(process.AnyOf(timeoutEv, el2.Event))

			if timeoutEv.Processed() && !el2.Event.Processed() {
				outData, err := s.HandleEvent(SensorWithUpdateData{TurnOn: false})
				if err != nil {
					slog.Warn("handler error", "id", s.ID, "err", err)
					continue
				}

				dto, _ := json.Marshal(outData)
				s.HandleOutDTO(dto)
				break
			}

			process.Wait(process.Timeout(s.Delay))
			nextData := el2.Item

			if !nextData.TurnOn {
				outData, err := s.HandleEvent(SensorWithUpdateData{TurnOn: false})
				if err != nil {
					slog.Warn("handler error", "id", s.ID, "err", err)
					continue
				}

				dto, _ := json.Marshal(outData)
				s.HandleOutDTO(dto)
				break
			}
		}
	}
}

func (s *SensorWithUpdate) HandleEvent(inData SensorWithUpdateData) (SensorWithUpdateData, error) {
	s.TurnedOn = inData.TurnOn

	return SensorWithUpdateData{
		Kind:   inData.Kind,
		TurnOn: s.TurnedOn,
	}, nil
}

// SensorWithoutUpdate - датчик окна (фиксирует открытие/закрытие окна)
// реализует интерфейс entities.EntityWithProcess.
type SensorWithoutUpdate struct {
	BaseDevice[SensorWithoutUpdateData]
	TurnedOn bool `json:"turn_on"`
}

type SensorWithoutUpdateData struct {
	Kind     string `json:"kind"`
	TurnedOn bool   `json:"turn_on"`
}

func NewSensorWithoutUpdate(data []byte, engineAPI engine.EnginePort) (*SensorWithoutUpdate, error) {
	var sensor SensorWithoutUpdate

	if err := json.Unmarshal(data, &sensor); err != nil {
		return nil, err
	}

	sensor.enginePort = engineAPI
	sensor.inStore = *simgo.NewStore[SensorWithoutUpdateData](engineAPI.GetSimulation())
	sensor.handler = sensor.HandleEvent

	return &sensor, nil
}

func (s *SensorWithoutUpdate) HandleInDTO(dto []byte) error {
	input := SensorWithoutUpdateData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)
	return nil
}

// HandleEvent реализует бизнес-логику устройства.
func (s *SensorWithoutUpdate) HandleEvent(inData SensorWithoutUpdateData) SensorWithoutUpdateData {
	s.TurnedOn = inData.TurnedOn

	return SensorWithoutUpdateData{
		Kind:     inData.Kind,
		TurnedOn: s.TurnedOn,
	}
}

// SensorWithIntStatus - датчик окна (фиксирует открытие/закрытие окна)
// реализует интерфейс entities.EntityWithProcess.
type SensorWithIntStatus struct {
	BaseDevice[SensorWithIntStatusData]
	Percents int `json:"percents"`
}

type SensorWithIntStatusData struct {
	Kind     string `json:"kind"`
	Percents int    `json:"percents"`
}

func NewSensorWithIntStatus(data []byte, engineAPI engine.EnginePort) (*SensorWithIntStatus, error) {
	var sensor SensorWithIntStatus

	if err := json.Unmarshal(data, &sensor); err != nil {
		return nil, err
	}

	sensor.enginePort = engineAPI
	sensor.inStore = *simgo.NewStore[SensorWithIntStatusData](engineAPI.GetSimulation())
	sensor.handler = sensor.HandleEvent

	return &sensor, nil
}

func (s *SensorWithIntStatus) HandleInDTO(dto []byte) error {
	input := SensorWithIntStatusData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)
	return nil
}

// HandleEvent реализует бизнес-логику устройства.
func (s *SensorWithIntStatus) HandleEvent(inData SensorWithIntStatusData) SensorWithIntStatusData {
	s.Percents = inData.Percents

	return SensorWithIntStatusData{
		Kind:     inData.Kind,
		Percents: s.Percents,
	}
}
