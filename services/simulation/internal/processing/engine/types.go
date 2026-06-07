package engine

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/fschuetz04/simgo"
)

// Engine определяет главный интерфейс для запуска и обработки симуляции
type Engine interface {
	// InitEntities инициализирует сущности и их зависимости.
	InitEntities(
		IDToEntity map[string]entities.Entity,
		IDToDependencies map[string][]api.EdgeDTO,
	)

	// InitProcesses инициализирует данные для процессов и запускает процессы.
	InitProcesses()

	// CheckCircleDependencies проверяет наличие циклических зависимостей среди сущностей.
	// Возвращает true, если цикл найден.
	CheckCircleDependencies() bool

	// SetFloor устанавливает поле для симуляции.
	SetFloor(floor *api.Floor)

	// GetInChan возвращает канал для входящих событий.
	GetInChan() chan api.EventDTO

	// GetOutChan возвращает канал для выходящих событий.
	GetOutChan() chan api.EventDTO

	// Step продвигает симуляционное время на dtSim вперёд.
	// Вызывается из Simulations.Tick после отправки всех входящих событий.
	Step()

	// CollectStep собирает результаты текущего тика и возвращает их клиенту.
	CollectStep(tick int) *api.SimulationStepPayload

	// Stop завершает симуляцию, закрывая канал входящих событий.
	Stop()

	// HandleEvent обрабатывает event по его entityID
	HandleEvent(event api.EventDTO)
}

// EnginePort определяет интерфейс для взаимодействия сущностей с движком
type EnginePort interface {
	// GetOutChan возвращает канал для отправки событий от сущностей к движку.
	GetOutChan() chan api.EventDTO

	// GetInChan возвращает канал для получения событий от движка к сущностям.
	GetInChan() chan api.EventDTO

	// GetSimulation возвращает объект симуляции для управления временем и процессами.
	GetSimulation() *simgo.Simulation

	// GetFloor возвращает текущее поле симуляции
	GetFloor() *api.Floor

	// GetRoomObservers возвращает список ID сущностей, которые наблюдают за данным roomID.
	GetRoomObservers(roomID string) []string

	// NotifyObservers отправляет уведомление всем наблюдателям за roomID с указанным kind и payload.
	NotifyObservers(roomID string, kind string, payload []byte)

	// DrainInChan читает события из входного канала.
	DrainInChan()
	GetEntity(id string) entities.Entity
}
