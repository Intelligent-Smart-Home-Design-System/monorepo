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
	panic("todo")
}

// TODO: функция для приема зависимостей между девайсами (либо сразу учитывать в GetEntities)

func (s *SimFetcher) GetEvents() ([]byte, error) {
	// TODO: запрос на новые данные
	panic("todo")
}

func (s *SimFetcher) GetField() ([]byte, error) {
	// TODO: запрос на новые данные
	panic("todo")
}
