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

	// HandleOutDTO обрабатывает исходящие данные, отправляет их в канал событий и тригерит ресиверов.
	HandleOutDTO(dto []byte)

	// GetProcessFunc возвращает функция процесс
	GetProcessFunc() func(process simgo.Process)
}

// Observer определяет интерфейс наблюдателя, который может наблюдать за событиями в комнате и реагировать на них.
type Observer interface {
	Entity

	// GetPosition возвращает координаты наблюдателя в комнате.
	GetPosition() (x, y float64)

	// GetObservedKinds возвращает список событий наблюдения.
	GetObservedKinds() []string // ["human:move"], ["fire:spread"] и тд
}

// Типы сущностей
const (
	TypeLamp                          = "lamp"
	TypeSmartLamp                     = "smartLamp"
	TypeSmartDimmer                   = "smartDimmer"
	TypeSwitcher                      = "switcher"
	TypeSensorWithUpdate              = "sensorWithUpdate"
	TypeSensorWithoutUpdate           = "sensorWithoutUpdate"
	TypeSensorWithIntStatus           = "sensorWithIntStatus"
	TypeRadiusMoveSensorWithUpdate    = "radiusMoveSensorWithUpdate"
	TypeRadiusMoveSensorWithoutUpdate = "radiusMoveSensorWithoutUpdate"
	TypeSiren                         = "siren"
	TypeWindow                        = "window"
	TypeDoor                          = "door"
	TypeSmartLock                     = "smartLock"
	TypeSmartDoorbell                 = "smartDoorbell"
	TypeSmartCurtains                 = "smartCurtains"
	TypeCamera                        = "camera"
	TypeAirConditioner                = "airConditioner"
	TypeThermostat                    = "thermostat"
	TypeSmartFloor                    = "smartFloor"
	TypeTV                            = "tv"
	TypeSubwoofer                     = "subwoofer"

	TypeHuman = "human"
)
