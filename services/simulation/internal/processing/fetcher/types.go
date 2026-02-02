package fetcher

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"

// Fetcher опреляет интрфейс для получения данных
type Fetcher interface {
	GetEntities() ([]api.EntityDTO, error)
	GetEvents() ([]api.EventInDTO, error)
	GetField() (api.FieldDTO, error)
}
