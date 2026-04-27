package engine

import (
	"errors"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
)

// Engine определяет главный интерфейс для запуска и обработки симуляции
type Engine interface {
	// InitEntities инициализирует сущности и их зависимости.
	InitEntities(
		IDToEntity map[string]entities.Entity,
		IDToDependencies map[string][]api.ActionDTO,
	)

	// InitProcesses инициализирует данные для процессов и запускает процессы.
	InitProcesses()

	// CheckCircleDependencies проверяет наличие циклических зависимостей среди сущностей.
	// Возвращает true, если цикл найден.
	CheckCircleDependencies() bool

	// SetField устанавливает поле для симуляции.
	SetField(simField *field.Field)

	// GetInChan возвращает канал для входящих событий.
	GetInChan() chan api.EventInDTO

	// Run запускает симуляцию, обрабатывая события из канала eventsInChan.
	Run() error

	// Step продвигает симуляционное время на dtSim вперёд.
	// Вызывается из Simulations.Tick после отправки всех входящих событий.
	Step()
 
	// CollectStep собирает результаты текущего тика и возвращает их клиенту.
	CollectStep(tick int) *api.SimulationStepPayload
 
	// Stop завершает симуляцию, закрывая канал входящих событий.
	Stop()

	// HandleEvent обрабатывает event по его entityID
	HandleEvent(event api.EventInDTO)

	// UpdateField обновляет состояние ячейки на поле. Если координаты некорректные, то возвращает ошибку.
	UpdateField(x, y int, cell field.Cell) error
}

// EnginePort определяет интерфейс для взаимодействия сущностей с движком
type EnginePort interface {
	UpdateField(x, y int, cell field.Cell) error
}

var (
	ErrorFieldInvalidParameterX = errors.New("invalid parameter x")
	ErrorFieldInvalidParameterY = errors.New("invalid parameter y")
)
