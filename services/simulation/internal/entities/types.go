package entities

import (
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/domain"
	"github.com/fschuetz04/simgo"
)

// Entity определяет главный интерфейс устройств
type Entity interface {
	// GetProcessFunc возвращает функция процесс
	GetProcessFunc() func(process simgo.Process, in domain.InData, out domain.OutData)

	// Process реализует функцию процесса устройства. Принимает данные через
	// inData и возвращает обработанные данные через outData
	Process(process simgo.Process, inData domain.InData, outData domain.OutData)

	// GetInDataStruct возвращает структуру для приема входных данных
	GetInDataStruct() domain.InData

	// GetOutDataStruct возвращает структуру для возврата обработанных данных
	GetOutDataStruct() domain.OutData

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
