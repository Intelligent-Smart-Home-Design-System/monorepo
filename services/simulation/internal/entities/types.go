package entities

// В пакете реализуется бизнес-логика сущностей

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/fschuetz04/simgo"
)

// Entity определяет интерфейс сущности без бизнес-логики
type Entity interface {
	// GetID возвращает ID сущности
	GetID() string

	// GetReceiversID возвращает сущности, которые данная сущность тригерит
	GetReceiversID() []string

	// SetReceivers устанавливает сущности, которые данная сущность тригерит
	SetReceivers(actions []api.ActionDTO)
}

// EntityWithProcess определяет интерфейс сущности с бизнес-логикой
type EntityWithProcess interface {
	Entity

	// HandleInDTO обрабатывает входящие данные и сохраняет их в хранилище сущности.
	HandleInDTO(dto []byte) error

	// HandleOutDTO обрабатывает исходящие данные, отправляет их в канал событий и тригерит ресиверов.
	HandleOutDTO(dto []byte)

	// GetProcessFunc возвращает функция процесс
	GetProcessFunc() func(process simgo.Process)

	// Process реализует функцию процесса устройства.
	Process(process simgo.Process)
}

const (
	TypeLamp         = "lamp"
	TypeLampSwitcher = "lampSwitcher"
)
