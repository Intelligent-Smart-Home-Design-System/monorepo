<<<<<<< HEAD
package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Датчики

// SensorWithUpdate - сенсор-переключатель с функцией обновления последнего действия.
// Позволяет обновлять действие до истечения таймаута, старое действие игнорируется.
type SensorWithUpdate struct {
	BaseDevice[SensorWithUpdateData]
	TurnOn  bool    `json:"turn_on"`
	Timeout float64 `json:"timeout"`
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

		if len(s.Receivers) != 0 {
			s.enginePort.DrainInChan()
		}

		for s.TurnOn {
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

		if len(s.Receivers) != 0 {
			s.enginePort.DrainInChan()
		}
	}
}

func (s *SensorWithUpdate) HandleEvent(inData SensorWithUpdateData) (SensorWithUpdateData, error) {
	s.TurnOn = inData.TurnOn

	return SensorWithUpdateData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}, nil
}

// SensorWithoutUpdate - датчик окна (фиксирует открытие/закрытие окна)
// реализует интерфейс entities.EntityWithProcess.
type SensorWithoutUpdate struct {
	BaseDevice[SensorWithoutUpdateData]
	TurnOn bool `json:"turn_on"`
}

type SensorWithoutUpdateData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
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
	s.TurnOn = inData.TurnOn

	return SensorWithoutUpdateData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
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

// RadiusMoveSensorWithUpdate — датчик с радиусом и таймаутом.
// Примеры: датчик движения (движение человека/устройства), датчик присутствия.
// Активируется когда объект входит в радиус, сбрасывается по таймауту или когда объект покидает радиус.
type RadiusMoveSensorWithUpdate struct {
	BaseDevice[RadiusSensorData]
	TurnOn  bool    `json:"turn_on"`
	Timeout float64 `json:"timeout"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Radius  float64 `json:"radius"`
}

type RadiusSensorData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewRadiusSensorWithUpdate(data []byte, engineAPI engine.EnginePort) (*RadiusMoveSensorWithUpdate, error) {
	var s RadiusMoveSensorWithUpdate
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	s.enginePort = engineAPI
	s.inStore = *simgo.NewStore[RadiusSensorData](engineAPI.GetSimulation())

	return &s, nil
}

func (s *RadiusMoveSensorWithUpdate) GetPosition() (float64, float64) {
	return s.X, s.Y
}

func (s *RadiusMoveSensorWithUpdate) HandleInDTO(dto []byte) error {
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

	s.Put(RadiusSensorData{
		Kind:   move.Kind,
		TurnOn: field.IsInRadius(s.X, s.Y, move.To.X, move.To.Y, s.Radius),
	})

	return nil
}

func (s *RadiusMoveSensorWithUpdate) GetProcessFunc() func(process simgo.Process) {
	return s.Process
}

func (s *RadiusMoveSensorWithUpdate) Process(process simgo.Process) {
	for {
		el := s.inStore.Get()
		process.Wait(el.Event)
		process.Wait(process.Timeout(s.Delay))

		inData := el.Item
		if !inData.TurnOn {
			continue
		}

		outData := s.HandleEvent(RadiusSensorData{Kind: inData.Kind, TurnOn: true})

		dto, _ := json.Marshal(outData)
		s.HandleOutDTO(dto)

		for s.TurnOn {
			timeoutEv := process.Timeout(s.Timeout)
			el2 := s.inStore.Get()

			process.Wait(process.AnyOf(timeoutEv, el2.Event))

			if timeoutEv.Processed() && !el2.Event.Processed() {
				outData := s.HandleEvent(RadiusSensorData{Kind: inData.Kind, TurnOn: false})

				dto, _ := json.Marshal(outData)
				s.HandleOutDTO(dto)

				break
			}

			process.Wait(process.Timeout(s.Delay))

			nextData := el2.Item

			if !nextData.TurnOn {
				outData := s.HandleEvent(RadiusSensorData{Kind: inData.Kind, TurnOn: false})

				dto, _ := json.Marshal(outData)
				s.HandleOutDTO(dto)

				break
			}
		}

		if len(s.Receivers) != 0 {
			s.enginePort.DrainInChan()
		}
	}
}

func (s *RadiusMoveSensorWithUpdate) HandleEvent(inData RadiusSensorData) RadiusSensorData {
	s.TurnOn = inData.TurnOn

	return RadiusSensorData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}
}

func (s *RadiusMoveSensorWithUpdate) GetObservedKinds() []string {
	return []string{"human:move", "device:move"}
}

// RadiusMoveSensorWithoutUpdate — датчик с радиусом без таймаута.
// Просто фиксирует факт попадания в радиус без сброса по таймауту.
type RadiusMoveSensorWithoutUpdate struct {
	BaseDevice[RadiusSensorData]
	TurnOn bool    `json:"turn_on"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

func NewRadiusSensorWithoutUpdate(data []byte, engineAPI engine.EnginePort) (*RadiusMoveSensorWithoutUpdate, error) {
	var s RadiusMoveSensorWithoutUpdate
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	s.enginePort = engineAPI
	s.inStore = *simgo.NewStore[RadiusSensorData](engineAPI.GetSimulation())
	s.handler = s.HandleEvent

	return &s, nil
}

func (s *RadiusMoveSensorWithoutUpdate) GetPosition() (float64, float64) {
	return s.X, s.Y
}

func (s *RadiusMoveSensorWithoutUpdate) HandleInDTO(dto []byte) error {
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

	s.Put(RadiusSensorData{
		Kind:   move.Kind,
		TurnOn: field.IsInRadius(s.X, s.Y, move.To.X, move.To.Y, s.Radius),
	})

	return nil
}

func (s *RadiusMoveSensorWithoutUpdate) HandleEvent(inData RadiusSensorData) RadiusSensorData {
	s.TurnOn = inData.TurnOn

	return RadiusSensorData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}
}

func (s *RadiusMoveSensorWithoutUpdate) GetObservedKinds() []string {
	return []string{"human:move", "device:move"}
}
=======
package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Датчики

// LightSwitchOffSensor - сенсор-переключатель
type LightSwitchOffSensor struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[LightSwitchOffSensorInData]

	ID        string   `json:"id"`
	TurnedOn  bool     `json:"turned_on"`
	Delay     float64  `json:"delay"`
	Timeout   float64  `json:"timeout"`
	Receivers []string `json:"receivers"`
}

type LightSwitchOffSensorInData struct {
	TurnOn bool `json:"turn_on"`
}

type LightSwitchOffSensorOutData struct {
	TurnOn bool `json:"turn_on"`
}

func NewLightSwitchOffSensor(data []byte, engineAPI engine.EnginePort) (*LightSwitchOffSensor, error) {
	var switcher LightSwitchOffSensor
	if err := json.Unmarshal(data, &switcher); err != nil {
		return nil, err
	}
	switcher.enginePort = engineAPI
	switcher.inStore = *simgo.NewStore[LightSwitchOffSensorInData](engineAPI.GetSimulation())
	return &switcher, nil
}

func (l *LightSwitchOffSensor) HandleInDTO(dto []byte) error {
	input := LightSwitchOffSensorInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}
	l.inStore.Put(input)
	return nil
}

func (l *LightSwitchOffSensor) HandleOutDTO(dto []byte) {
	outData := api.EventOutDTO{
		EntityID: l.ID,
		Payload:  dto,
	}

	l.enginePort.GetOutChan() <- outData

	for _, receiverID := range l.Receivers {
		l.enginePort.GetInChan() <- api.EventInDTO{
			EntityID: receiverID,
			Payload:  dto,
		}
	}
}

func (l *LightSwitchOffSensor) GetProcessFunc() func(process simgo.Process) {
	return l.Process
}

func (l *LightSwitchOffSensor) Process(process simgo.Process) {
	for {
		el := l.inStore.Get()
		process.Wait(el.Event)
		process.Wait(process.Timeout(l.getReactionDelay()))

		inData := el.Item
		if !inData.TurnOn {
			continue
		}

		outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: true})
		dataLamp, _ := json.Marshal(outData)
		l.HandleOutDTO(dataLamp)

		for l.TurnedOn {
			timeoutEv := process.Timeout(l.Timeout)
			el2 := l.inStore.Get()

			process.Wait(process.AnyOf(timeoutEv, el2.Event))

			if timeoutEv.Processed() && !el2.Event.Processed() {
				outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: false})
				dto, _ := json.Marshal(outData)
				l.HandleOutDTO(dto)
				break
			}

			process.Wait(process.Timeout(l.getReactionDelay()))
			nextData := el2.Item

			if !nextData.TurnOn {
				outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: false})
				dto, _ := json.Marshal(outData)
				l.HandleOutDTO(dto)
				break
			}
		}
	}
}

func (l *LightSwitchOffSensor) HandleEvent(inData LightSwitchOffSensorInData) LightSwitchOffSensorOutData {
	l.TurnedOn = inData.TurnOn

	out := LightSwitchOffSensorOutData{
		TurnOn: l.TurnedOn,
	}

	return out
}

func (l *LightSwitchOffSensor) GetID() string {
	return l.ID
}

func (l *LightSwitchOffSensor) getReactionDelay() float64 {
	return l.Delay
}

func (l *LightSwitchOffSensor) GetReceiversID() []string {
	return l.Receivers
}

func (l *LightSwitchOffSensor) SetReceivers(actions []api.EdgeDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ToID
	}

	l.Receivers = receivers
}
>>>>>>> 4bf54f8 (hz)
