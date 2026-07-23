package sources

import (
	"context"
	"fmt"
	"net/url"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	dnsParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/dns"
	dnsScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/dns"
)

// DNS wires scrapers/dns and parsers/dns into the unified Source interface.
type DNS struct {
	Base
	log zerolog.Logger
}

func newDNS(cfg config.Config, log zerolog.Logger) Source {
	ua := cfg.Dns.UserAgent
	if ua == "" {
		ua = cfg.Scraping.UserAgent
	}
	s := dnsScraper.NewScraper(log, cfg.Scraping.Timeout, cfg.Scraping.Proxy, ua, cfg.Dns.BrowserUserMode)
	return DNS{
		Base: Base{name: domain.SourceDns, scraper: s},
		log:  log.With().Str("source", domain.SourceDns).Logger(),
	}
}

func (s DNS) Warmup(ctx context.Context) error {
	return s.Scraper().(*dnsScraper.Scraper).Warmup(ctx)
}

// BootstrapDiscovery — step A: discovery_seeds + search_queries from config.
func (DNS) BootstrapDiscovery(cfg config.Config) []TaskSeed {
	seed, _ := discoveryBootstrapModes(cfg, domain.SourceDns)
	if !seed {
		return nil
	}
	var out []TaskSeed
	for _, seedURL := range cfg.Dns.DiscoverySeeds {
		out = append(out, TaskSeed{
			Source:   domain.SourceDns,
			PageType: domain.PageTypeDiscovery,
			URL:      seedURL,
		})
	}
	maxPages := cfg.Dns.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}
	for _, query := range cfg.Dns.SearchQueries {
		for page := 1; page <= maxPages; page++ {
			searchURL := fmt.Sprintf(
				"https://www.dns-shop.ru/search/?q=%s&page=%d",
				url.QueryEscape(query), page,
			)
			out = append(out, TaskSeed{
				Source:   domain.SourceDns,
				PageType: domain.PageTypeDiscovery,
				URL:      searchURL,
			})
		}
	}
	return out
}

// ExpandDiscovery — step B: parsers/dns.RunDiscoveryBFS over catalog seeds.
func (s DNS) ExpandDiscovery(ctx context.Context, cfg config.Config) ([]TaskSeed, error) {
	seed, _ := discoveryBootstrapModes(cfg, domain.SourceDns)
	if !seed || len(cfg.Dns.DiscoverySeeds) == 0 {
		return nil, nil
	}
	var repo MemTaskRepo
	stats, err := dnsParser.RunDiscoveryBFS(
		ctx, s.log, cfg.Scraping, cfg.Dns, cfg.Dns.DiscoverySeeds, &repo, s.Scraper(),
	)
	if err != nil {
		return nil, fmt.Errorf("dns discovery bfs: %w", err)
	}
	s.log.Info().
		Int("fetches", stats.Fetches).
		Int("hubs_visited", stats.HubsVisited).
		Int("categories_created", stats.CategoriesCreated).
		Msg("dns discovery bfs complete")
	return repo.Seeds, nil
}

// DiscoveryScrapeTypes — step C: discovery (optional) + category from DB.
func (DNS) DiscoveryScrapeTypes(cfg config.Config) []domain.PageType {
	_, db := discoveryBootstrapModes(cfg, domain.SourceDns)
	var types []domain.PageType
	if db {
		types = append(types, domain.PageTypeDiscovery)
	}
	return append(types, domain.PageTypeCategory)
}
