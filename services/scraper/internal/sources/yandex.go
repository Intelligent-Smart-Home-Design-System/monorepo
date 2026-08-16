package sources

import (
	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	yandexScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/yandex"
)

type yandexSource struct {
	Base
	log zerolog.Logger
}

func newYandex(cfg config.Config, log zerolog.Logger) Source {
	s := yandexScraper.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.Proxy, cfg.Scraping.RateLimitRps)
	return yandexSource{
		Base: Base{name: domain.SourceYandex, scraper: s},
		log:  log.With().Str("source", domain.SourceYandex).Logger(),
	}
}

func (yandexSource) BootstrapListing(cfg config.Config) []TaskSeed {
	url := cfg.Yandex.SupportedZigbeeDevicesURL
	if url == "" {
		return nil
	}
	return []TaskSeed{{
		Source:   domain.SourceYandex,
		PageType: domain.PageTypeCompatibility,
		URL:      url,
	}}
}
