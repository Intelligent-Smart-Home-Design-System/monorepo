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
	Opened bool `json:"opened"`
}

type WindowData struct {
	Kind   string `json:"kind"`
	Opened bool   `json:"opened"`
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
	w.Opened = inData.Opened

	return WindowData{
		Kind:   inData.Kind,
		Opened: w.Opened,
	}
}

// Door — исполнительное устройство (актуатор).
// Получает команды от датчика/логики и меняет состояние двери.
type Door struct {
	BaseDevice[DoorData]
	Opened bool `json:"opened"`
}

type DoorData struct {
	Kind   string `json:"kind"`
	Opened bool   `json:"opened"`
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
	d.Opened = inData.Opened

	return DoorData{
		Kind:   inData.Kind,
		Opened: d.Opened,
	}
}

// SmartLock — исполнительное устройство (замок).
// Управляет состоянием "заблокирован / разблокирован".
type SmartLock struct {
	BaseDevice[LockData]
	Locked bool `json:"locked"`
}

type LockData struct {
	Kind   string `json:"kind"`
	Locked bool   `json:"locked"`
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
	l.Locked = inData.Locked

	return LockData{
		Kind:   inData.Kind,
		Locked: l.Locked,
	}
}

// SmartDoorbell — исполнительное устройство (умный дверной звонок).
// Получает событие нажатия и генерирует сигнал/уведомление.
type SmartDoorbell struct {
	BaseDevice[DoorbellData]
	Ringing bool `json:"ringing"`
}

type DoorbellData struct {
	Kind string `json:"kind"`
	Ring bool   `json:"ring"`
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
	d.Ringing = inData.Ring

	return DoorbellData{
		Kind: inData.Kind,
		Ring: d.Ringing,
	}
}

// SmartCurtains — исполнительное устройство (умные шторы).
// Управляет положением штор: 0 = закрыты, 100 = открыты.
type SmartCurtains struct {
	BaseDevice[CurtainsData]
	Position int `json:"position"` // 0-100
}

type CurtainsData struct {
	Kind     string `json:"kind"`
	Position int    `json:"position"`
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
	pos := inData.Position

	if pos < 0 {
		pos = 0
	}
	if pos > 100 {
		pos = 100
	}

	c.Position = pos

	return CurtainsData{
		Kind:     inData.Kind,
		Position: c.Position,
	}
}
