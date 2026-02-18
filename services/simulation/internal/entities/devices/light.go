package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Освещение

// Lamp реализует интерфейс entities.Entity
type Lamp struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[LampInData]

	ID        string   `json:"id"`
	TurnedOn  bool     `json:"turned_on"`
	Delay     float64  `json:"delay"`
	Receivers []string `json:"receivers"`
}

type LampInData struct {
	TurnOn bool `json:"turn_on"`
}

type LampOutData struct {
	Time float64 `json:"time"`
}

func NewLamp(data []byte, engineAPI engine.EnginePort) (*Lamp, error) {
	var lamp Lamp
	if err := json.Unmarshal(data, &lamp); err != nil {
		return nil, err
	}

	lamp.enginePort = engineAPI

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

func (l *Lamp) HandleOutDTO(out LampOutData) error {
	dataLamp, err := json.Marshal(out)
	if err != nil {
		return err
	}

	outData := api.EventOutDTO{
		EntityID: l.ID,
		Type:     entities.TypeLamp,
		Info:     dataLamp,
	}

	l.enginePort.GetOutChan() <- outData

	return nil
}

func (l *Lamp) GetProcessFunc() func(process simgo.Process) {
	return l.Process
}

func (l *Lamp) Process(process simgo.Process) {
	for {
		storeElement := l.inStore.Get()
		inData := storeElement.Item
		event := storeElement.Event

		process.Wait(event)                    // ждем пока прийдет событие в store с
		process.Wait(process.Timeout(l.Delay)) // учитывая задержку

		outData := l.HandleEvent(process, inData)
		err := l.HandleOutDTO(outData)
		slog.Warn("error in event handle", "error", err)
	}
}

// HandleEvent реализует бизнес логику обработки сущности
func (l *Lamp) HandleEvent(process simgo.Process, inData LampInData) LampOutData {
	l.TurnedOn = inData.TurnOn

	out := LampOutData{
		Time: process.Simulation.Now(),
	}

	return out
}

func (l *Lamp) GetID() string {
	return l.ID
}

func (l *Lamp) GetReceiversID() []string {
	return l.Receivers
}

func (l *Lamp) GetReactionDelay() float64 {
	return l.Delay
}
