package engine

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
)

// Engine определяет главный интерфейс для запуска и обработки симуляции
type Engine interface {
	InitEntities(IDToEntity map[string]entities.Entity)
	InitProcesses()
	GetQueue() chan config.EventDTO
	Run() error
	HandleEvent(event config.EventDTO)
}
