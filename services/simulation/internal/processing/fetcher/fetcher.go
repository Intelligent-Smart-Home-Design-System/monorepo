package fetcher

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"

// TODO: прием данных

// SimFetcher реализует Fetcher
type SimFetcher struct {
}

func (s *SimFetcher) GetEntities() []*config.EventDTO {
	// TODO: запрос на новые данные
	panic("todo")
}

// TODO: функция для приема зависимостей между девайсами (либо сразу учитывать в GetEntities)

func (s *SimFetcher) GetUpdates() []byte {
	// TODO: запрос на новые данные
	panic("todo")
}

func (s *SimFetcher) GetField() []byte {
	// TODO: запрос на новые данные
	panic("todo")
}
