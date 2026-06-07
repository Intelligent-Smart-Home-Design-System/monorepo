<<<<<<< HEAD
package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Lamp реализует интерфейс entities.EntityWithProcess.
type Lamp struct {
	BaseDevice[LampData]
	TurnOn bool `json:"turn_on"`
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
	l.TurnOn = inData.TurnOn

	return LampData{
		Kind:   inData.Kind,
		TurnOn: l.TurnOn,
	}
}

type SmartLamp struct {
	BaseDevice[SmartLampData]
	Percents int `json:"percents"`
}

type SmartLampData struct {
	Kind     string `json:"kind"`
	Percents int    `json:"percents"`
}

func NewSmartLamp(data []byte, engineAPI engine.EnginePort) (*SmartLamp, error) {
	var lamp SmartLamp
	if err := json.Unmarshal(data, &lamp); err != nil {
		return nil, err
	}

	lamp.enginePort = engineAPI
	lamp.inStore = *simgo.NewStore[SmartLampData](engineAPI.GetSimulation())
	lamp.handler = lamp.HandleEvent

	return &lamp, nil
}

func (s *SmartLamp) HandleInDTO(dto []byte) error {
	input := SmartLampData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	s.Put(input)

	return nil
}

// HandleEvent реализует бизнес-логику устройства.
// Возвращает обработанные данные.
func (s *SmartLamp) HandleEvent(inData SmartLampData) SmartLampData {
	s.Percents = inData.Percents

	return SmartLampData{
		Kind:     inData.Kind,
		Percents: s.Percents,
	}
}

// SmartDimmer (управление светом — используется для акцентного освещения и сцен)
// реализует интерфейс entities.EntityWithProcess.
type SmartDimmer struct {
	BaseDevice[DimmerData]
	Percents int `json:"percents"` // 0-100
}

type DimmerData struct {
	Kind     string `json:"kind"`
	Percents int    `json:"percents"`
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
	percents := inData.Percents

	if percents < 0 {
		percents = 0
	}

	if percents > 100 {
		percents = 100
	}

	d.Percents = percents

	out := DimmerData{
		Kind:     inData.Kind,
		Percents: d.Percents,
	}

	return out
}
=======
package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Lamp реализует интерфейс entities.EntityWithProcess.
type Lamp struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[LampInData]

	ID        string   `json:"id"`
	TurnedOn  bool     `json:"turned_on"`
	Delay     float64  `json:"delay"`
	Receivers []string `json:"receivers"`
}

// TODO: решить какие структуры использовать (SH-37, вопрос 1 фев 15:06)
type LampInData struct {
	TurnOn bool `json:"turn_on"`
}

type LampOutData struct {
	TurnOn bool `json:"turn_on"`
}

func NewLamp(data []byte, engineAPI engine.EnginePort) (*Lamp, error) {
	var lamp Lamp
	if err := json.Unmarshal(data, &lamp); err != nil {
		return nil, err
	}

	lamp.enginePort = engineAPI
	lamp.inStore = *simgo.NewStore[LampInData](engineAPI.GetSimulation())
	return &lamp, nil
}

func (l *Lamp) HandleInDTO(dto []byte) error {
	input := LampInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	l.inStore.Put(input)

	return nil
}

func (l *Lamp) HandleOutDTO(dto []byte) {
    outData := api.EventOutDTO{
        EntityID: l.ID,
        Payload:  dto,
    }
    l.enginePort.GetOutChan() <- outData
}

func (l *Lamp) GetProcessFunc() func(process simgo.Process) {
	return l.Process
}

func (l *Lamp) Process(process simgo.Process) {
	for {
		storeElement := l.inStore.Get()
		event := storeElement.Event

        process.Wait(event)
        process.Wait(process.Timeout(l.getReactionDelay()))

		inData := storeElement.Item
		outData := l.HandleEvent(inData)
		dto, err := json.Marshal(outData)
        l.HandleOutDTO(dto)
		slog.Warn("error in event handle", "error", err, "entity_id", l.ID)
	}
}

// HandleEvent реализует бизнес-логику устройства.
// Возвращает обработанные данные.
func (l *Lamp) HandleEvent(inData LampInData) LampOutData {
	l.TurnedOn = inData.TurnOn

	out := LampOutData{
		TurnOn: l.TurnedOn,
	}

	return out
}

func (l *Lamp) GetID() string {
	return l.ID
}

func (l *Lamp) getReactionDelay() float64 {
	return l.Delay
}

func (l *Lamp) GetReceiversID() []string {
	return l.Receivers
}

func (l *Lamp) SetReceivers(actions []api.EdgeDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ToID
	}

	l.Receivers = receivers
}
>>>>>>> 4bf54f8 (hz)
