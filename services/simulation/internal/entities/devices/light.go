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
		Patch:    dataLamp,
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
		event := storeElement.Event

		process.Wait(event)
		process.Wait(process.Timeout(l.getReactionDelay()))

		inData := storeElement.Item
		outData := l.HandleEvent(inData)
		err := l.HandleOutDTO(outData)
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
