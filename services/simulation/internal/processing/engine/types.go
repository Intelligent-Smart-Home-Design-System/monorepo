package engine

import (
	"context"
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

	// GetOutChan возвращает канал для исходящих событий.
	GetOutChan() chan api.EventOutDTO

	// Run запускает симуляцию, обрабатывая события из канала eventsInChan.
	// Если контекст отменен или канал закрыт, то симуляция завершается.
	Run(ctx context.Context) error

	// HandleEvent обрабатывает event по его entityID
	HandleEvent(event api.EventInDTO)

	// UpdateField обновляет состояние ячейки на поле. Если координаты некорректные, то возвращает ошибку.
	UpdateField(x, y int, cell field.Cell) error
}

// EnginePort определяет интерфейс для взаимодействия сущностей с движком
type EnginePort interface {
	UpdateField(x, y int, cell field.Cell) error
	GetOutChan() chan api.EventOutDTO
}

var (
	ErrorFieldInvalidParameterX = errors.New("invalid parameter x")
	ErrorFieldInvalidParameterY = errors.New("invalid parameter y")
)
