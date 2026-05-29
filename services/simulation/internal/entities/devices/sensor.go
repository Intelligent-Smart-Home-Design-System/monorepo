package devices

import (
	"encoding/json"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// Датчики

// LightSwitchOffSensor - сенсор-переключатель
type LightSwitchOffSensor struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[LightSwitchOffSensorInData]

	ID        string   `json:"id"`
	TurnedOn  bool     `json:"turned_on"`
	Delay     float64  `json:"delay"`
	Timeout   float64  `json:"timeout"`
	Receivers []string `json:"receivers"`
}

type LightSwitchOffSensorInData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

type LightSwitchOffSensorOutData struct {
	Kind   string `json:"kind"`
	TurnOn bool   `json:"turn_on"`
}

func NewLightSwitchOffSensor(data []byte, engineAPI engine.EnginePort) (*LightSwitchOffSensor, error) {
	var switcher LightSwitchOffSensor
	if err := json.Unmarshal(data, &switcher); err != nil {
		return nil, err
	}
	switcher.enginePort = engineAPI
	switcher.inStore = *simgo.NewStore[LightSwitchOffSensorInData](engineAPI.GetSimulation())
	return &switcher, nil
}

func (l *LightSwitchOffSensor) HandleInDTO(dto []byte) error {
	input := LightSwitchOffSensorInData{}
	if err := json.Unmarshal(dto, &input); err != nil {
		return err
	}
	l.inStore.Put(input)
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
		el := l.inStore.Get()
		process.Wait(el.Event)
		process.Wait(process.Timeout(l.getReactionDelay()))

		inData := el.Item
		if !inData.TurnOn {
			continue
		}

		outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: true})
		dataLamp, _ := json.Marshal(outData)
		l.HandleOutDTO(dataLamp)

		for l.TurnedOn {
			timeoutEv := process.Timeout(l.Timeout)
			el2 := l.inStore.Get()

			process.Wait(process.AnyOf(timeoutEv, el2.Event))

			if timeoutEv.Processed() && !el2.Event.Processed() {
				outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: false})
				dto, _ := json.Marshal(outData)
				l.HandleOutDTO(dto)
				break
			}

			process.Wait(process.Timeout(l.getReactionDelay()))
			nextData := el2.Item

			if !nextData.TurnOn {
				outData := l.HandleEvent(LightSwitchOffSensorInData{TurnOn: false})
				dto, _ := json.Marshal(outData)
				l.HandleOutDTO(dto)
				break
			}
		}
	}
}

func (l *LightSwitchOffSensor) HandleEvent(inData LightSwitchOffSensorInData) LightSwitchOffSensorOutData {
	l.TurnedOn = inData.TurnOn

	out := LightSwitchOffSensorOutData{
		Kind:   inData.Kind,
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
