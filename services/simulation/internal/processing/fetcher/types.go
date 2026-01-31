package fetcher

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"

// Fetcher опреляет интрфейс для получения данных
type Fetcher interface {
	GetEntities() ([]config.EntityDTO, error)
	GetEvents() ([]byte, error)
	GetField() ([]byte, error)
}
