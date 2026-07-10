package sources

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	sprutScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/sprut"
)

func newSprutScraperImpl(cfg config.Config) *sprutScraper.Scraper {
	return sprutScraper.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.Proxy, cfg.Scraping.UserAgent)
}
