package entities

// В пакете реализуется бизнес-логика сущностей

import (
	"time"

	"github.com/fschuetz04/simgo"
)

// Entity определяет главный интерфейс устройств
type Entity interface {
	// GetProcessFunc возвращает функция процесс
	GetProcessFunc() func(process simgo.Process)

	// Process реализует функцию процесса устройства. Принимает данные через
	// inData и возвращает обработанные данные через outData
	Process(process simgo.Process)

	// GetID возвращает ID сущности
	GetID() string
}

// Device опреляет главный интерфейс девайса. Наследует интерфейс Entity
// и добавляет собственные методы.
type Device interface {
	Entity

	GetType() string
	GetReactionDelay() time.Duration
}
