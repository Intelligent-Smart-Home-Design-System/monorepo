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

// SensorWithUpdateData - данные для SensorWithUpdate.
type SensorWithUpdateData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewSensorWithUpdate конструктор для SensorWithUpdate.
func NewSensorWithUpdate(data []byte, engineAPI engine.EnginePort) (*SensorWithUpdate, error) {
	var switcher SensorWithUpdate
	if err := json.Unmarshal(data, &switcher); err != nil {
		return nil, err
	}

	switcher.enginePort = engineAPI
	switcher.inStore = *simgo.NewStore[SensorWithUpdateData](engineAPI.GetSimulation())

	return &switcher, nil
}

// HandleInDTO обрабатывает входящие данные, сохраняет их в хранилище и запускает процесс обработки.
func (s *SensorWithUpdate) HandleInDTO(dto []byte) error {
	input := SensorWithUpdateData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)

	return nil
}

// GetProcessFunc возвращает функцию процесса для SensorWithUpdate.
func (s *SensorWithUpdate) GetProcessFunc() func(process simgo.Process) {
	return s.Process
}

// Process реализует бизнес-логику устройства с учетом обновления действия до истечения таймаута.
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

// HandleEvent реализует бизнес-логику устройства, обновляет состояние и возвращает данные для отправки.
func (s *SensorWithUpdate) HandleEvent(inData SensorWithUpdateData) (SensorWithUpdateData, error) {
	s.TurnOn = inData.TurnOn

	return SensorWithUpdateData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}, nil
}

// SensorWithoutUpdate - булевый датчик с функцией без обновления последнего действия.
// реализует интерфейс entities.EntityWithProcess.
type SensorWithoutUpdate struct {
	BaseDevice[SensorWithoutUpdateData]
	TurnOn bool `json:"turn_on"`
}

// SensorWithoutUpdateData - данные для SensorWithoutUpdate.
type SensorWithoutUpdateData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewSensorWithoutUpdate конструктор для SensorWithoutUpdate.
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

// HandleInDTO обрабатывает входящие данные, сохраняет их в хранилище и запускает процесс обработки.
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

// SensorWithIntStatus - целочисленный датчик без функции обновления последнего действия.
// реализует интерфейс entities.EntityWithProcess.
type SensorWithIntStatus struct {
	BaseDevice[SensorWithIntStatusData]
	Percents int `json:"percents"`
}

// SensorWithIntStatusData - данные для SensorWithIntStatus.
type SensorWithIntStatusData struct {
	Kind     string `json:"kind"`
	Percents int    `json:"percents"`
}

// NewSensorWithIntStatus конструктор для SensorWithIntStatus.
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

// HandleInDTO обрабатывает входящие данные, сохраняет их в хранилище и запускает процесс обработки.
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
// Имеет функцию обновления последнего действия, позволяет обновлять действие до истечения таймаута, старое действие игнорируется.
type RadiusMoveSensorWithUpdate struct {
	BaseDevice[RadiusSensorData]
	TurnOn  bool    `json:"turn_on"`
	Timeout float64 `json:"timeout"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Radius  float64 `json:"radius"`
}

// RadiusSensorData - данные для RadiusMoveSensorWithUpdate и RadiusMoveSensorWithoutUpdate.
type RadiusSensorData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewRadiusSensorWithUpdate конструктор для RadiusMoveSensorWithUpdate.
func NewRadiusSensorWithUpdate(data []byte, engineAPI engine.EnginePort) (*RadiusMoveSensorWithUpdate, error) {
	var s RadiusMoveSensorWithUpdate
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	s.enginePort = engineAPI
	s.inStore = *simgo.NewStore[RadiusSensorData](engineAPI.GetSimulation())

	return &s, nil
}

// GetPosition возвращает координаты датчика.
func (s *RadiusMoveSensorWithUpdate) GetPosition() (float64, float64) {
	return s.X, s.Y
}

// HandleInDTO обрабатывает входящие данные, сохраняет их в хранилище и запускает процесс обработки.
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

// GetProcessFunc возвращает функцию процесса для RadiusMoveSensorWithUpdate.
func (s *RadiusMoveSensorWithUpdate) GetProcessFunc() func(process simgo.Process) {
	return s.Process
}

// Process реализует бизнес-логику устройства с учетом обновления действия до истечения таймаута.
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

// HandleEvent реализует бизнес-логику устройства, обновляет состояние и возвращает данные для отправки.
func (s *RadiusMoveSensorWithUpdate) HandleEvent(inData RadiusSensorData) RadiusSensorData {
	s.TurnOn = inData.TurnOn

	return RadiusSensorData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}
}

// GetObservedKinds возвращает список видов событий, которые наблюдает датчик.
func (s *RadiusMoveSensorWithUpdate) GetObservedKinds() []string {
	return []string{"human:move", "device:move"}
}

// RadiusMoveSensorWithoutUpdate - датчик с радиусом без таймаута.
// Просто фиксирует факт попадания в радиус без сброса по таймауту.
type RadiusMoveSensorWithoutUpdate struct {
	BaseDevice[RadiusSensorData]
	TurnOn bool    `json:"turn_on"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

// NewRadiusSensorWithoutUpdate конструктор для RadiusMoveSensorWithoutUpdate.
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

// GetPosition возвращает координаты датчика.
func (s *RadiusMoveSensorWithoutUpdate) GetPosition() (float64, float64) {
	return s.X, s.Y
}

// HandleInDTO обрабатывает входящие данные, сохраняет их в хранилище и запускает процесс обработки.
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

// HandleEvent реализует бизнес-логику устройства, обновляет состояние и возвращает данные для отправки.
func (s *RadiusMoveSensorWithoutUpdate) HandleEvent(inData RadiusSensorData) RadiusSensorData {
	s.TurnOn = inData.TurnOn

	return RadiusSensorData{
		Kind:   inData.Kind,
		TurnOn: s.TurnOn,
	}
}

// GetObservedKinds возвращает список видов событий, которые наблюдает датчик.
func (s *RadiusMoveSensorWithoutUpdate) GetObservedKinds() []string {
	return []string{"human:move", "device:move"}
}
