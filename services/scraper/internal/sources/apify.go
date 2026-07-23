package sources

import (
	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	apifyScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/apify"
)

type apifySource struct {
	Base
	log     zerolog.Logger
	queries []string
}

func newApify(cfg config.Config, log zerolog.Logger) Source {
	s := apifyScraper.NewScraper(
		cfg.Scraping.Timeout,
		cfg.Scraping.Proxy,
		cfg.Apify.APIKey,
		cfg.Apify.ActorID,
		cfg.Apify.Region,
		cfg.Apify.MaxItems,
	)
	return apifySource{
		Base:    Base{name: domain.SourceApifyYandexMarket, scraper: s},
		log:     log.With().Str("source", domain.SourceApifyYandexMarket).Logger(),
		queries: cfg.Apify.SearchQueries,
	}
}

func (a apifySource) BootstrapDiscovery(cfg config.Config) []TaskSeed {
	seeds := make([]TaskSeed, 0, len(a.queries))
	for _, query := range a.queries {
		seeds = append(seeds, TaskSeed{
			Source:   domain.SourceApifyYandexMarket,
			PageType: domain.PageTypeDiscovery,
			URL:      query,
		})
	}
	return seeds
}
