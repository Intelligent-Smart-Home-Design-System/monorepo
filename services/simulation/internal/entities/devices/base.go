package devices

import (
	"encoding/json"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// BaseDevice - базовая структура для устройств, которая реализует интерфейс Entity и ProcessEntity.
type BaseDevice[T any] struct {
	enginePort engine.EnginePort
	inStore    simgo.Store[T]

	handler func(T) T

	ID        string   `json:"id"`
	Delay     float64  `json:"delay"`
	Receivers []string `json:"receivers"`
}

// HandleOutDTO отправляет DTO в выходной канал движка и во входные каналы всех получателей.
func (b *BaseDevice[T]) HandleOutDTO(dto []byte) {
	b.enginePort.GetOutChan() <- api.EventDTO{
		EntityID: b.ID,
		Payload:  dto,
	}

	for _, r := range b.Receivers {
		b.enginePort.GetInChan() <- api.EventDTO{
			EntityID: r,
			Payload:  dto,
		}
	}
}

// Process - основной процесс устройства, который обрабатывает входящие события, вызывает handler и отправляет результаты.
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

// GetProcessFunc возвращает функцию Process для использования в симуляции.
func (b *BaseDevice[T]) GetProcessFunc() func(simgo.Process) {
	return b.Process
}

// Put кладет элемент в simgo.Store для отправки в Process устройства.
func (b *BaseDevice[T]) Put(item T) {
	b.inStore.Put(item)
}

// SetReceivers устанавливает получателей для устройства на основе переданных действий.
func (b *BaseDevice[T]) SetReceivers(actions []api.EdgeDTO) {
	b.Receivers = make([]string, len(actions))
	for i, a := range actions {
		b.Receivers[i] = a.ToID
	}
}

// GetID возвращает ID устройства.
func (b *BaseDevice[T]) GetID() string {
	return b.ID
}

// GetReceiversID возвращает список ID получателей устройства.
func (b *BaseDevice[T]) GetReceiversID() []string {
	return b.Receivers
}
