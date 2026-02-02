package fetcher

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"

// TODO: прием данных

// SimFetcher реализует Fetcher
type SimFetcher struct {
}

func NewSimFetcher() *SimFetcher {
	return &SimFetcher{}
}

func (s *SimFetcher) GetEntities() ([]api.EntityDTO, error) {
	// TODO: запрос на новые данные
	return make([]api.EntityDTO, 0), nil
}

// TODO: функция для приема зависимостей между девайсами (либо сразу учитывать в GetEntities)

func (s *SimFetcher) GetEvents() ([]api.EventInDTO, error) {
	// TODO: запрос на новые данные
	return make([]api.EventInDTO, 0), nil
}

func (s *SimFetcher) GetField() (api.FieldDTO, error) {
	// TODO: запрос на новые данные
	return api.FieldDTO{}, nil
}
