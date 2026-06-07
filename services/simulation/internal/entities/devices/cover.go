package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Окна / ворота

// Window — исполнительное устройство.
type Window struct {
	BaseDevice[WindowData]
	TurnOn bool `json:"turn_on"`
}

// WindowData — данные для управления окном.
type WindowData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewWindow - конструктор окна. Парсит JSON-конфигурацию и инициализирует устройство.
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

// HandleInDTO - парсит входящие данные и кладет их в simgo.Store устройства для последующей обработки.
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

// Door — исполнительное устройство.
type Door struct {
	BaseDevice[DoorData]
	TurnOn bool `json:"turn_on"`
}

// DoorData - данные для управления дверью.
type DoorData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewDoor - конструктор двери. Парсит JSON-конфигурацию и инициализирует устройство.
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

// HandleInDTO - парсит входящие данные и кладет их в simgo.Store устройства для последующей обработки.
func (d *Door) HandleInDTO(dto []byte) error {
	input := DoorData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	d.Put(input)

	return nil
}

// HandleEvent - бизнес-логика двери.
func (d *Door) HandleEvent(inData DoorData) DoorData {
	d.TurnOn = inData.TurnOn

	return DoorData{
		Kind:   inData.Kind,
		TurnOn: d.TurnOn,
	}
}

// SmartLock — исполнительное устройство.
type SmartLock struct {
	BaseDevice[LockData]
	TurnOn bool `json:"turn_on"`
}

// LockData - данные для управления замком.
type LockData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewSmartLock - конструктор замка. Парсит JSON-конфигурацию и инициализирует устройство.
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

// HandleInDTO - парсит входящие данные и кладет их в simgo.Store устройства для последующей обработки.
func (l *SmartLock) HandleInDTO(dto []byte) error {
	input := LockData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	l.Put(input)

	return nil
}

// HandleEvent - бизнес-логика замка.
func (l *SmartLock) HandleEvent(inData LockData) LockData {
	l.TurnOn = inData.TurnOn

	return LockData{
		Kind:   inData.Kind,
		TurnOn: l.TurnOn,
	}
}

// SmartDoorbell - исполнительное устройство.
type SmartDoorbell struct {
	BaseDevice[DoorbellData]
	TurnOn bool `json:"turn_on"`
}

// DoorbellData - данные для управления дверным звонком.
type DoorbellData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

// NewSmartDoorbell - конструктор дверного звонка. Парсит JSON-конфигурацию и инициализирует устройство.
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

// HandleInDTO - парсит входящие данные и кладет их в simgo.Store устройства для последующей обработки.
func (d *SmartDoorbell) HandleInDTO(dto []byte) error {
	input := DoorbellData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	d.Put(input)

	return nil
}

// HandleEvent - бизнес-логика дверного звонка.
func (d *SmartDoorbell) HandleEvent(inData DoorbellData) DoorbellData {
	d.TurnOn = inData.TurnOn

	return DoorbellData{
		Kind:   inData.Kind,
		TurnOn: d.TurnOn,
	}
}

// SmartCurtains - исполнительное устройство (умные шторы).
// Управляет положением штор: 0 = закрыты, 100 = открыты.
type SmartCurtains struct {
	BaseDevice[CurtainsData]
	Percents int `json:"percents"`
}

// CurtainsData - данные для управления шторами.
type CurtainsData struct {
	Kind     string `json:"kind"`
	Percents int    `json:"percents"`
}

// NewSmartCurtains - конструктор штор. Парсит JSON-конфигурацию и инициализирует устройство.
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

// HandleInDTO - парсит входящие данные и кладет их в simgo.Store устройства для последующей обработки.
func (c *SmartCurtains) HandleInDTO(dto []byte) error {
	input := CurtainsData{}

	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	c.Put(input)

	return nil
}

// HandleEvent - бизнес-логика штор.
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

// SmartFloor - умный тёплый пол (вкл/выкл + полигон зоны обогрева).
type SmartFloor struct {
	BaseDevice[SmartFloorData]
	TurnOn      bool         `json:"turn_on"`
	Temperature int          `json:"temperature"`
	Area        [][2]float64 `json:"area"` // полигон зоны обогрева
}

// SmartFloorData - данные для управления тёплым полом.
type SmartFloorData struct {
	Kind        string `json:"kind"`
	TurnOn      *bool  `json:"turn_on"`
	Temperature *int   `json:"temperature"`
}

// NewSmartFloor - конструктор тёплого пола. Парсит JSON-конфигурацию и инициализирует устройство.
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

// HandleInDTO - парсит входящие данные и кладет их в simgo.Store устройства для последующей обработки.
func (sf *SmartFloor) HandleInDTO(dto []byte) error {
	input := SmartFloorData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	sf.Put(input)

	return nil
}

// HandleEvent - бизнес-логика тёплого пола.
func (sf *SmartFloor) HandleEvent(inData SmartFloorData) SmartFloorData {
	if inData.TurnOn != nil {
		sf.TurnOn = *inData.TurnOn
	}

	if inData.Temperature != nil {
		sf.Temperature = *inData.Temperature
	}

	return SmartFloorData{
		Kind:        inData.Kind,
		TurnOn:      &sf.TurnOn,
		Temperature: &sf.Temperature,
	}
}
