package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Переключатели / розетки

type LampSwitcher struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[LampSwitcherInData]

	ID        string   `json:"id"`
	TurnedOn  bool     `json:"turned_on"`
	Delay     float64  `json:"delay"`
	Receivers []string `json:"receivers"`
}

// TODO: решить какие структуры использовать (SH-37, вопрос 1 фев 15:06)
type LampSwitcherInData struct {
	TurnOn bool `json:"turn_on"`
}

type LampSwitcherOutData struct {
	TurnOn bool `json:"turn_on"`
}

func NewLampSwitcher(data []byte, engineAPI engine.EnginePort) (*LampSwitcher, error) {
	var lampSwitcher LampSwitcher
	if err := json.Unmarshal(data, &lampSwitcher); err != nil {
		return nil, err
	}

	lampSwitcher.enginePort = engineAPI

	return &lampSwitcher, nil
}

func (l *LampSwitcher) HandleInDTO(dto []byte) error {
	input := LampSwitcherInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	l.inStore.Put(input)

	return nil
}

func (l *LampSwitcher) HandleOutDTO(out LampSwitcherOutData) error {
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

func (l *LampSwitcher) GetProcessFunc() func(process simgo.Process) {
	return l.Process
}

func (l *LampSwitcher) Process(process simgo.Process) {
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

func (l *LampSwitcher) HandleEvent(inData LampSwitcherInData) LampSwitcherOutData {
	l.TurnedOn = inData.TurnOn

	out := LampSwitcherOutData{
		TurnOn: l.TurnedOn,
	}

	return out
}

func (l *LampSwitcher) GetID() string {
	return l.ID
}

func (l *LampSwitcher) getReactionDelay() float64 {
	return l.Delay
}

func (l *LampSwitcher) GetReceiversID() []string {
	return l.Receivers
}

func (l *LampSwitcher) SetReceivers(actions []api.EdgeDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ToID
	}

	l.Receivers = receivers
}
