package entities

// В пакете реализуется бизнес-логика сущностей

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/fschuetz04/simgo"
)

// Entity определяет интерфейс сущности без бизнес-логики
type Entity interface {
	// GetID возвращает ID сущности
	GetID() string

	// GetReceiversID возвращает сущности, который данная сущность тригерит
	GetReceiversID() []string

	// SetReceivers устанавливает сущности, которые данная сущность тригерит
	SetReceivers(actions []api.ActionDTO) []string

	// GetLocation возвращает координаты местонахождения сущности на поле
	GetLocation() field.Cell
}

// EntityWithProcess определяет интерфейс сущности с бизнес-логикой
type EntityWithProcess interface {
	Entity

	// GetProcessFunc возвращает функция процесс
	GetProcessFunc() func(process simgo.Process)

	// Process реализует функцию процесса устройства. Принимает данные через
	// inData и возвращает обработанные данные через outData
	Process(process simgo.Process)

	HandleInDTO(dto []byte) error

	GetOutCh() chan []byte
}

const (
	TypeLamp = "lamp"
)
