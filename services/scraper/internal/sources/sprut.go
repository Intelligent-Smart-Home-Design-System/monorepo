package sources

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	sprutParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/sprut"
	sprutScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/sprut"
)

// Sprut wires scrapers/sprut and parsers/sprut into the unified Source interface.
// Same three-step discovery shape as DNS (sources/dns.go): root/hub catalog pages are
// walked in memory (ExpandDiscovery), only product-grid pages are persisted as page_type=category.
type Sprut struct {
	Base
	log zerolog.Logger
}

func newSprut(cfg config.Config, log zerolog.Logger) Source {
	s := sprutScraper.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.Proxy, cfg.Scraping.UserAgent)
	return Sprut{
		Base: Base{name: domain.SourceSprut, scraper: s},
		log:  log.With().Str("source", domain.SourceSprut).Logger(),
	}
}

// BootstrapDiscovery — step A: root catalog seeds from config (e.g. https://sprut.ai/catalog/section).
func (Sprut) BootstrapDiscovery(cfg config.Config) []TaskSeed {
	seed, _ := discoveryBootstrapModes(cfg, domain.SourceSprut)
	if !seed {
		return nil
	}
	var out []TaskSeed
	for _, seedURL := range cfg.Sprut.DiscoverySeeds {
		out = append(out, TaskSeed{
			Source:   domain.SourceSprut,
			PageType: domain.PageTypeDiscovery,
			URL:      seedURL,
		})
	}
	return out
}

// ExpandDiscovery — step B: parsers/sprut.RunDiscoveryBFS over catalog seeds.
func (s Sprut) ExpandDiscovery(ctx context.Context, cfg config.Config) ([]TaskSeed, error) {
	seed, _ := discoveryBootstrapModes(cfg, domain.SourceSprut)
	if !seed || len(cfg.Sprut.DiscoverySeeds) == 0 {
		return nil, nil
	}
	var repo MemTaskRepo
	stats, err := sprutParser.RunDiscoveryBFS(
		ctx, s.log, cfg.Scraping, cfg.Sprut, cfg.Sprut.DiscoverySeeds, &repo, s.Scraper(),
	)
	if err != nil {
		return nil, fmt.Errorf("sprut discovery bfs: %w", err)
	}
	s.log.Info().
		Int("fetches", stats.Fetches).
		Int("hubs_visited", stats.HubsVisited).
		Int("categories_created", stats.CategoriesCreated).
		Msg("sprut discovery bfs complete")
	return repo.Seeds, nil
}

// DiscoveryScrapeTypes — step C: discovery (optional) + category from DB.
func (Sprut) DiscoveryScrapeTypes(cfg config.Config) []domain.PageType {
	_, db := discoveryBootstrapModes(cfg, domain.SourceSprut)
	var types []domain.PageType
	if db {
		types = append(types, domain.PageTypeDiscovery)
	}
	return append(types, domain.PageTypeCategory)
}
