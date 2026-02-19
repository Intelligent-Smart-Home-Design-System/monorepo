package engine

import (
	"context"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
)

// Engine определяет главный интерфейс для запуска и обработки симуляции
type Engine interface {
	InitEntities(
		IDToEntity map[string]entities.Entity,
		IDToDependencies map[string][]api.ActionDTO,
	)
	InitProcesses()
	CheckCircleDependencies() bool
	GetInChan() chan api.EventInDTO
	GetOutChan() chan api.EventOutDTO
	Run(ctx context.Context) error
	HandleEvent(event api.EventInDTO)
	SetField(simField *field.Field)
	UpdateField(x, y int, cell field.Cell) error
}

type EnginePort interface {
	UpdateField(x, y int, cell field.Cell) error
	GetOutChan() chan api.EventOutDTO
}
