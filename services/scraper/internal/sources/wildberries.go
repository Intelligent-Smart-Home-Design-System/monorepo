package sources

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	wbScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
)

const wbDiscoveryPrefix = "wildberries://discovery/"

// Wildberries wires scrapers/wildberries into the unified Source interface.
type Wildberries struct {
	Base
	log zerolog.Logger
}

func newWildberries(cfg config.Config, log zerolog.Logger) Source {
	s := wbScraper.NewScraper(
		log,
		cfg.Scraping.Timeout,
		cfg.Scraping.Proxy,
		cfg.Scraping.WBCardBasket,
		cfg.Scraping.WBRPS,
		cfg.Scraping.WBSessionPath,
		cfg.Wildberries.Discovery.URLTemplate,
		cfg.Wildberries.Discovery.MaxPages,
		cfg.Wildberries.BrowserUserMode,
		cfg.Wildberries.BrowserProfileDir,
	)
	return Wildberries{
		Base: Base{name: domain.SourceWildberries, scraper: s},
		log:  log.With().Str("source", domain.SourceWildberries).Logger(),
	}
}

func (s Wildberries) Warmup(ctx context.Context) error {
	return s.Scraper().(*wbScraper.Scraper).Warmup(ctx)
}

// Close releases the dedicated Chrome profile (required between isolated WB smoke steps).
func (s Wildberries) Close() {
	if wb, ok := s.Scraper().(*wbScraper.Scraper); ok {
		wb.Close()
	}
}

// BootstrapDiscovery — step A: discovery_text_queries → wildberries://discovery/{query}.
func (s Wildberries) BootstrapDiscovery(cfg config.Config) []TaskSeed {
	seed, _ := discoveryBootstrapModes(cfg, domain.SourceWildberries)
	if !seed || len(cfg.Wildberries.Discovery.DiscoveryTextQueries) == 0 {
		return nil
	}
	out := make([]TaskSeed, 0, len(cfg.Wildberries.Discovery.DiscoveryTextQueries))
	for _, query := range cfg.Wildberries.Discovery.DiscoveryTextQueries {
		out = append(out, TaskSeed{
			Source:   domain.SourceWildberries,
			PageType: domain.PageTypeDiscovery,
			URL:      fmt.Sprintf("%s%s", wbDiscoveryPrefix, query),
		})
	}
	return out
}

// ExpandDiscovery — step B: not used (WB resolves queries at scrape time via URL template).
func (Wildberries) ExpandDiscovery(context.Context, config.Config) ([]TaskSeed, error) {
	return nil, nil
}

// DiscoveryScrapeTypes — step C: fetch discovery snapshots from DB when db bootstrap is on.
func (Wildberries) DiscoveryScrapeTypes(cfg config.Config) []domain.PageType {
	_, db := discoveryBootstrapModes(cfg, domain.SourceWildberries)
	if !db {
		return nil
	}
	return []domain.PageType{domain.PageTypeDiscovery}
}

func (s Wildberries) CleanupDiscovery(ctx context.Context, repo TaskRepo, cfg config.Config, enabled bool) error {
	_ = ctx
	if !enabled {
		return nil
	}
	allTasks, err := repo.GetTasks("", "", 0)
	if err != nil {
		return fmt.Errorf("get tasks for cleanup: %w", err)
	}
	if len(cfg.Wildberries.Discovery.DiscoveryTextQueries) > 0 {
		keep := make(map[string]bool, len(cfg.Wildberries.Discovery.DiscoveryTextQueries))
		for _, q := range cfg.Wildberries.Discovery.DiscoveryTextQueries {
			keep[q] = true
		}
		for _, t := range allTasks {
			if t.Source != domain.SourceWildberries || t.PageType != domain.PageTypeDiscovery {
				continue
			}
			if keep[strings.TrimPrefix(t.URL, wbDiscoveryPrefix)] {
				continue
			}
			if err := repo.DeleteTaskByID(t.ID); err != nil {
				s.log.Error().Err(err).Int("task_id", t.ID).Msg("failed to delete stale discovery task")
			}
		}
		return nil
	}
	for _, t := range allTasks {
		if t.Source == domain.SourceWildberries && t.PageType == domain.PageTypeDiscovery {
			_ = repo.DeleteTaskByID(t.ID)
		}
	}
	return nil
}

func (Wildberries) BootstrapListing(cfg config.Config) []TaskSeed {
	url := cfg.Wildberries.Category.CategoryURL
	if url == "" {
		return nil
	}
	return []TaskSeed{{
		Source:   domain.SourceWildberries,
		PageType: domain.PageTypeCategory,
		URL:      url,
	}}
}
