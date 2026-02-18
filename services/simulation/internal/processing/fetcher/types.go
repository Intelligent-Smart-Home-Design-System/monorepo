package fetcher

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"

// Fetcher опреляет интрфейс для получения данных
type Fetcher interface {
	// GetSimulationsID возвращает список ID симуляций.
	GetSimulationsID() []string

	// GetEntities возвращает данные о сущностях на основе ID симуляции.
	GetEntities() (map[string][]api.EntityDTO, error)

	// GetDependencies возвращает данные о зависимостях между сущностями на основе ID симуляции.
	GetDependencies() (map[string]map[string][]api.ActionDTO, error)

	// GetEvents возвращает данные о событиях на основе ID симуляции.
	GetEvents() (map[string][]api.EventInDTO, error)

	// GetFields возвращает данные о поле для симуляции на основе ID симуляции.
	GetFields() (map[string]api.FieldDTO, error)
}
