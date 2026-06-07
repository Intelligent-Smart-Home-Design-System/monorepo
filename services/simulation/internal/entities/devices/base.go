package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

type BaseDevice[T any] struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[T]

	handler func(T) T

	ID        string   `json:"id"`
	Delay     float64  `json:"delay"`
	Receivers []string `json:"receivers"`
}

func (b *BaseDevice[T]) HandleOutDTO(dto []byte) {
	b.enginePort.GetOutChan() <- api.EventOutDTO{
		EntityID: b.ID,
		Payload:  dto,
	}

	for _, r := range b.Receivers {
		b.enginePort.GetInChan() <- api.EventInDTO{
			EntityID: r,
			Payload:  dto,
		}
	}
}

func (b *BaseDevice[T]) Process(process simgo.Process) {
	for {
		storeElement := b.inStore.Get()

		event := storeElement.Event
		process.Wait(event)
		process.Wait(process.Timeout(b.Delay))

		out := b.handler(storeElement.Item)

		dto, err := json.Marshal(out)
		if err != nil {
			slog.Warn("marshal error", "id", b.ID, "err", err)
			continue
		}

		b.HandleOutDTO(dto)

		if len(b.Receivers) != 0 {
			b.enginePort.DrainInChan()
		}
	}
}

func (b *BaseDevice[T]) GetProcessFunc() func(simgo.Process) {
	return b.Process
}

// Put кладет элемент в simgo.Store для отправки в Process устройства.
func (b *BaseDevice[T]) Put(item T) {
	b.inStore.Put(item)
}

func (b *BaseDevice[T]) SetReceivers(actions []api.EdgeDTO) {
	b.Receivers = make([]string, len(actions))
	for i, a := range actions {
		b.Receivers[i] = a.ToID
	}
}

func (b *BaseDevice[T]) GetID() string {
	return b.ID
}

func (b *BaseDevice[T]) GetReceiversID() []string {
	return b.Receivers
}
