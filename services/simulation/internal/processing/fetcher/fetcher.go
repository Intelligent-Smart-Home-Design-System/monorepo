package fetcher

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"

// TODO: прием данных

// SimFetcher реализует Fetcher
type SimFetcher struct {
}

func NewSimFetcher() *SimFetcher {
	return &SimFetcher{}
}

func (s *SimFetcher) GetEntities() ([]config.EntityDTO, error) {
	// TODO: запрос на новые данные
	return make([]config.EntityDTO, 0), nil
}

// TODO: функция для приема зависимостей между девайсами (либо сразу учитывать в GetEntities)

func (s *SimFetcher) GetEvents() ([]config.EventInDTO, error) {
	// TODO: запрос на новые данные
	return make([]config.EventInDTO, 0), nil
}

func (s *SimFetcher) GetField() (config.FieldDTO, error) {
	// TODO: запрос на новые данные
	return config.FieldDTO{}, nil
}
