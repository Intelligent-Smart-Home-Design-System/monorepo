package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Переключатели / розетки

// LampSwitcher - стандартный переключатель
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

	lampSwitcher.inStore = *simgo.NewStore[LampSwitcherInData](engineAPI.GetSimulation())

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

func (l *LampSwitcher) HandleOutDTO(dto []byte) {
	outData := api.EventOutDTO{
		EntityID: l.ID,
		Info:     dto,
	}

	l.enginePort.GetOutChan() <- outData

	for _, receiverID := range l.Receivers {
		l.enginePort.GetInChan() <- api.EventInDTO{
			EntityID: receiverID,
			Info:     dto, // TODO: подумать, то ли мы передаем
		}
	}
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

		dataLamp, err := json.Marshal(outData)
		if err != nil {
			slog.Warn("cannot marshal out data", "error", err, "entity_id", l.ID)
		}
		l.HandleOutDTO(dataLamp)
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

func (l *LampSwitcher) SetReceivers(actions []api.ActionDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ID
	}

	l.Receivers = receivers
}

type LightSwitchOffSensor struct {
	enginePort  engine.EnginePort
	simEvent    *simgo.Event
	simCancelEv *simgo.Event
	data        LightSwitchOffSensorInData

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
	switcher.simEvent = engineAPI.GetSimulation().Event()
	switcher.simCancelEv = engineAPI.GetSimulation().Event()

	return &switcher, nil
}

func (l *LightSwitchOffSensor) HandleInDTO(dto []byte) error {
	input := LightSwitchOffSensorInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	oldSimEvent := l.simEvent
	oldCancelEv := l.simCancelEv

	l.simCancelEv = l.enginePort.GetSimulation().Event()
	l.simEvent = l.enginePort.GetSimulation().Event()
	l.data = input

	oldCancelEv.Trigger()
	oldSimEvent.Trigger()

	return nil
}

func (l *LightSwitchOffSensor) HandleOutDTO(dto []byte) {
	outData := api.EventOutDTO{
		EntityID: l.ID,
		Info:     dto,
	}

	l.enginePort.GetOutChan() <- outData

	for _, receiverID := range l.Receivers {
		l.enginePort.GetInChan() <- api.EventInDTO{
			EntityID: receiverID,
			Info:     dto, // TODO: подумать, то ли мы передаем
		}
	}
}

func (l *LightSwitchOffSensor) GetProcessFunc() func(process simgo.Process) {
	return l.Process
}

func (l *LightSwitchOffSensor) Process(process simgo.Process) {
	for {
		event := l.simEvent
		process.Wait(event)

		for {
			process.Wait(process.Timeout(l.getReactionDelay()))
			inData := l.data
			cancelEv := l.simCancelEv

			if inData.TurnOn == false {
				timeoutEv := process.Timeout(l.Timeout)
				process.Wait(process.AnyOf(timeoutEv, cancelEv))

				if cancelEv.Processed() {
					continue
				}
			}

			outData := l.HandleEvent(inData)
			dataLamp, err := json.Marshal(outData)
			if err != nil {
				slog.Warn("cannot marshal out data", "error", err, "entity_id", l.ID)
			}
			l.HandleOutDTO(dataLamp)
			break
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

func (l *LightSwitchOffSensor) SetReceivers(actions []api.ActionDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ID
	}

	l.Receivers = receivers
}
