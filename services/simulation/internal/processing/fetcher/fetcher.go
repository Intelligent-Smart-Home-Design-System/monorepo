package fetcher

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"

// TODO: прием данных

// SimFetcher реализует Fetcher
type SimFetcher struct {
}

func NewSimFetcher() *SimFetcher {
	return &SimFetcher{}
}

// GetSimulationsID возвращает список ID симуляций.
func (s *SimFetcher) GetSimulationsID() []string {
	return make([]string, 0)
}

// GetEntities возвращает данные о сущностях на основе ID симуляции.
func (s *SimFetcher) GetEntities() (map[string][]api.EntityDTO, error) {
	// TODO: запрос на новые данные
	return make(map[string][]api.EntityDTO), nil
}

// GetDependencies возвращает данные о зависимостях между сущностями на основе ID симуляции.
func (s *SimFetcher) GetDependencies() (map[string]api.ActionDTO, error) {
	// TODO: запрос на получение зависимостей
	return make(map[string]api.ActionDTO), nil
}

// GetEvents возвращает данные о событиях на основе ID симуляции.
func (s *SimFetcher) GetEvents() (map[string][]api.EventInDTO, error) {
	// TODO: запрос на новые данные
	return make(map[string][]api.EventInDTO), nil
}

// GetFields возвращает данные о поле для симуляции на основе ID симуляции.
func (s *SimFetcher) GetFields() (map[string]api.FieldDTO, error) {
	// TODO: запрос на новые данные
	return map[string]api.FieldDTO{}, nil
}
