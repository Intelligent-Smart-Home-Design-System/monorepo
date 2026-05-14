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
	SetReceivers(actions []api.EdgeDTO)
}

// EntityWithProcess определяет интерфейс сущности с бизнес-логикой
type EntityWithProcess interface {
	Entity

	// HandleInDTO обрабатывает входящие данные и сохраняет их в хранилище сущности.
	HandleInDTO(dto []byte) error

	// HandleOutDTO обрабатывает исходящие данные и отправляет их в канал событий.
	HandleOutDTO(out any) error

	// GetProcessFunc возвращает функция процесс
	GetProcessFunc() func(process simgo.Process)

	// Process реализует функцию процесса устройства.
	Process(process simgo.Process)

	// GetOutCh возвращает канал для отправки данных о событиях.
	GetOutCh() chan []byte
}

const (
	TypeLamp         = "lamp"
	TypeLampSwitcher = "lampSwitcher"
)
