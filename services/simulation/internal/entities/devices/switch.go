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

func (l *LampSwitcher) SetReceivers(actions []api.EdgeDTO) {
	receivers := make([]string, len(actions))
	for i, action := range actions {
		receivers[i] = action.ToID
	}

	l.Receivers = receivers
}

// LightSwitchOffSensor - сенсор-переключатель
type LightSwitchOffSensor struct {
	enginePort engine.EnginePort
	turnOnEv   *simgo.Event
	turnOffEv  *simgo.Event
	data       LightSwitchOffSensorInData

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
	switcher.turnOnEv = engineAPI.GetSimulation().Event()
	switcher.turnOffEv = engineAPI.GetSimulation().Event()

	return &switcher, nil
}

func (l *LightSwitchOffSensor) HandleInDTO(dto []byte) error {
	input := LightSwitchOffSensorInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}

	l.data = input

	if input.TurnOn {
		l.turnOnEv.Trigger()
		l.turnOnEv = l.enginePort.GetSimulation().Event()
	} else {
		l.turnOffEv.Trigger()
		l.turnOffEv = l.enginePort.GetSimulation().Event()
	}

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
		turnOnEv := l.turnOnEv
		turnOffEv := l.turnOffEv
		process.Wait(process.AnyOf(turnOnEv, turnOffEv))

		if turnOffEv.Processed() {
			continue
		}

		turnOffEv = l.turnOffEv
		turnOnEv = l.turnOnEv
		process.Wait(process.Timeout(l.getReactionDelay()))

		outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: true})
		dataLamp, _ := json.Marshal(outData)
		l.HandleOutDTO(dataLamp)

		for l.TurnedOn {
			turnOnEv = l.turnOnEv
			process.Wait(process.AnyOf(turnOffEv, turnOnEv))

			if turnOnEv.Processed() {
				continue
			}

			turnOffEv = l.turnOffEv
			turnOnEv = l.turnOnEv
			process.Wait(process.Timeout(l.getReactionDelay()))

			if turnOnEv.Processed() {
				continue
			}

			for {
				timeoutEv := process.Timeout(l.Timeout)
				process.Wait(process.AnyOf(timeoutEv, turnOffEv, turnOnEv))

				if turnOnEv.Processed() {
					turnOnEv = l.turnOnEv
					break
				}

				if turnOffEv.Processed() {
					turnOffEv = l.turnOffEv
					turnOnEv = l.turnOnEv
					process.Wait(process.Timeout(l.getReactionDelay()))
					if turnOnEv.Processed() {
						break
					}
					continue
				}

				outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: false})
				dataLamp, _ := json.Marshal(outData)
				l.HandleOutDTO(dataLamp)
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
