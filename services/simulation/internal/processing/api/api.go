package api

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
)

// api процессинга для работы во вне (например в бизнес-логики устройств)

type EngineAPI interface {
	UpdateField(x, y int, cell field.Cell) error
	GetOutChan() chan config.EventOutDTO
}
