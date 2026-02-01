package engine

import (
	"context"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
)

// Engine определяет главный интерфейс для запуска и обработки симуляции
type Engine interface {
	InitEntities(IDToBaseEntity map[string]entities.Entity)
	InitProcesses()
	GetInChan() chan config.EventInDTO
	GetOutChan() chan config.EventOutDTO
	Run(ctx context.Context) error
	HandleEvent(event config.EventInDTO)
	SetField(simField *field.Field)
	UpdateField(x, y int, cell field.Cell) error
}
