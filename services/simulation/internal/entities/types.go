package entities

// В пакете реализуется бизнес-логика сущностей

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/fschuetz04/simgo"
)

// Entity определяет интерфейс сущности без бизнес-логики
type Entity interface {
	// GetID возвращает ID сущности
	GetID() string

	// GetReceiversID возвращает сущности, который данная сущность тригерит
	GetReceiversID() []string

	// GetLocation возвращает координаты местонахождения сущности на поле
	GetLocation() config.Cell
}

// EntityWithProcess определяет интерфейс сущности с бизнес-логикой
type EntityWithProcess interface {
	Entity

	// GetProcessFunc возвращает функция процесс
	GetProcessFunc() func(process simgo.Process)

	// Process реализует функцию процесса устройства. Принимает данные через
	// inData и возвращает обработанные данные через outData
	Process(process simgo.Process)

	// SendEvent присылает новый ивент для обработки
	SendEvent(dto config.EventDTO)
}
