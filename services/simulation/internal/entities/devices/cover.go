package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Окна / ворота

// Window реализует интерфейс entities.EntityWithProcess.
type Window struct {
	BaseDevice[WindowData]
	TurnOn bool `json:"turn_on"`
}

type WindowData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewWindow(data []byte, engineAPI engine.EnginePort) (*Window, error) {
	var window Window

	if err := json.Unmarshal(data, &window); err != nil {
		return nil, err
	}

	window.enginePort = engineAPI
	window.inStore = *simgo.NewStore[WindowData](engineAPI.GetSimulation())
	window.handler = window.HandleEvent

	return &window, nil
}

func (w *Window) HandleInDTO(dto []byte) error {
	input := WindowData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	w.Put(input)

	return nil
}

// HandleEvent реализует бизнес-логику окна.
func (w *Window) HandleEvent(inData WindowData) WindowData {
	w.TurnOn = inData.TurnOn

	return WindowData{
		Kind:   inData.Kind,
		TurnOn: w.TurnOn,
	}
}

// Door — исполнительное устройство (актуатор).
// Получает команды от датчика/логики и меняет состояние двери.
type Door struct {
	BaseDevice[DoorData]
	TurnOn bool `json:"turn_on"`
}

type DoorData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewDoor(data []byte, engineAPI engine.EnginePort) (*Door, error) {
	var d Door

	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	d.enginePort = engineAPI
	d.inStore = *simgo.NewStore[DoorData](engineAPI.GetSimulation())
	d.handler = d.HandleEvent

	return &d, nil
}

func (d *Door) HandleInDTO(dto []byte) error {
	input := DoorData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	d.Put(input)

	return nil
}

// HandleEvent — бизнес-логика двери.
func (d *Door) HandleEvent(inData DoorData) DoorData {
	d.TurnOn = inData.TurnOn

	return DoorData{
		Kind:   inData.Kind,
		TurnOn: d.TurnOn,
	}
}

// SmartLock — исполнительное устройство (замок).
// Управляет состоянием "заблокирован / разблокирован".
type SmartLock struct {
	BaseDevice[LockData]
	TurnOn bool `json:"turn_on"`
}

type LockData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSmartLock(data []byte, engineAPI engine.EnginePort) (*SmartLock, error) {
	var l SmartLock

	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}

	l.enginePort = engineAPI
	l.inStore = *simgo.NewStore[LockData](engineAPI.GetSimulation())
	l.handler = l.HandleEvent

	return &l, nil
}

func (l *SmartLock) HandleInDTO(dto []byte) error {
	input := LockData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	l.Put(input)

	return nil
}

// HandleEvent — бизнес-логика замка.
func (l *SmartLock) HandleEvent(inData LockData) LockData {
	l.TurnOn = inData.TurnOn

	return LockData{
		Kind:   inData.Kind,
		TurnOn: l.TurnOn,
	}
}

// SmartDoorbell — исполнительное устройство (умный дверной звонок).
// Получает событие нажатия и генерирует сигнал/уведомление.
type SmartDoorbell struct {
	BaseDevice[DoorbellData]
	TurnOn bool `json:"turn_on"`
}

type DoorbellData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSmartDoorbell(data []byte, engineAPI engine.EnginePort) (*SmartDoorbell, error) {
	var d SmartDoorbell

	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	d.enginePort = engineAPI
	d.inStore = *simgo.NewStore[DoorbellData](engineAPI.GetSimulation())
	d.handler = d.HandleEvent

	return &d, nil
}

func (d *SmartDoorbell) HandleInDTO(dto []byte) error {
	input := DoorbellData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	d.Put(input)

	return nil
}

// HandleEvent — бизнес-логика дверного звонка.
func (d *SmartDoorbell) HandleEvent(inData DoorbellData) DoorbellData {
	d.TurnOn = inData.TurnOn

	return DoorbellData{
		Kind:   inData.Kind,
		TurnOn: d.TurnOn,
	}
}

// SmartCurtains — исполнительное устройство (умные шторы).
// Управляет положением штор: 0 = закрыты, 100 = открыты.
type SmartCurtains struct {
	BaseDevice[CurtainsData]
	Percents int `json:"percents"`
}

type CurtainsData struct {
	Kind     string `json:"kind"`
	Percents int    `json:"percents"`
}

func NewSmartCurtains(data []byte, engineAPI engine.EnginePort) (*SmartCurtains, error) {
	var c SmartCurtains

	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	c.enginePort = engineAPI
	c.inStore = *simgo.NewStore[CurtainsData](engineAPI.GetSimulation())
	c.handler = c.HandleEvent

	return &c, nil
}

func (c *SmartCurtains) HandleInDTO(dto []byte) error {
	input := CurtainsData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	c.Put(input)

	return nil
}

// HandleEvent — бизнес-логика штор.
func (c *SmartCurtains) HandleEvent(inData CurtainsData) CurtainsData {
	pos := inData.Percents

	if pos < 0 {
		pos = 0
	}

	if pos > 100 {
		pos = 100
	}

	c.Percents = pos

	return CurtainsData{
		Kind:     inData.Kind,
		Percents: c.Percents,
	}
}

// SmartFloor — умный тёплый пол (вкл/выкл + полигон зоны обогрева).
type SmartFloor struct {
	BaseDevice[SmartFloorData]
	TurnOn bool         `json:"turn_on"`
	Area   [][2]float64 `json:"area"` // полигон зоны обогрева
}

type SmartFloorData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewSmartFloor(data []byte, engineAPI engine.EnginePort) (*SmartFloor, error) {
	var sf SmartFloor
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}

	sf.enginePort = engineAPI
	sf.inStore = *simgo.NewStore[SmartFloorData](engineAPI.GetSimulation())
	sf.handler = sf.HandleEvent

	return &sf, nil
}

func (sf *SmartFloor) HandleInDTO(dto []byte) error {
	input := SmartFloorData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	sf.Put(input)

	return nil
}

func (sf *SmartFloor) HandleEvent(inData SmartFloorData) SmartFloorData {
	sf.TurnOn = inData.TurnOn

	return SmartFloorData{
		Kind:   inData.Kind,
		TurnOn: sf.TurnOn,
	}
}
