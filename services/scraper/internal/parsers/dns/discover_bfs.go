package dns

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	dnsScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/dns"
)

// DiscoverStats summarizes in-memory DNS catalog BFS during scrape --discovery.
type DiscoverStats struct {
	HubsVisited      int
	CategoriesCreated int
	InMemoryEnqueued int
}

type categoryTaskWriter interface {
	CreateTask(source, pageType, url string) error
}

// RunDiscoveryBFS walks hub pages in memory and persists only category URLs to tracked_pages.
// Snapshots are saved later by the scrape worker (same as WB: CreateTask then worker scrape).
func RunDiscoveryBFS(
	ctx context.Context,
	logger zerolog.Logger,
	scraping config.ScrapingConfig,
	dnsCfg config.DnsConfig,
	seeds []string,
	tasks categoryTaskWriter,
) (DiscoverStats, error) {
	scraper := dnsScraper.NewScraper(scraping.Timeout, scraping.Proxy, userAgent(scraping, dnsCfg))
	stats := DiscoverStats{}

	queue := append([]string{}, seeds...)
	seen := make(map[string]bool, len(seeds))

	for len(queue) > 0 {
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}
		url := queue[0]
		queue = queue[1:]
		if seen[url] {
			continue
		}
		seen[url] = true

		if err := processPage(ctx, logger, scraper, scraping.RateLimitRps, url, tasks, &stats, seen, &queue); err != nil {
			logger.Warn().Err(err).Str("url", url).Msg("dns bfs: page failed")
		}
	}

	logger.Info().
		Int("hubs_visited", stats.HubsVisited).
		Int("categories_created", stats.CategoriesCreated).
		Int("in_memory_enqueued", stats.InMemoryEnqueued).
		Msg("dns bfs finished")

	return stats, nil
}

func userAgent(scraping config.ScrapingConfig, dnsCfg config.DnsConfig) string {
	if dnsCfg.UserAgent != "" {
		return dnsCfg.UserAgent
	}
	return scraping.UserAgent
}

func processPage(
	ctx context.Context,
	logger zerolog.Logger,
	scraper *dnsScraper.Scraper,
	rateLimitRPS float64,
	pageURL string,
	tasks categoryTaskWriter,
	stats *DiscoverStats,
	seen map[string]bool,
	queue *[]string,
) error {
	throttle(rateLimitRPS)

	result, err := scraper.Scrape(ctx, domain.ScrapeTask{URL: pageURL})
	if err != nil {
		return err
	}
	if len(result.Resources) == 0 {
		return fmt.Errorf("empty scrape result for %s", pageURL)
	}
	html, err := parser.FindFile(toArchiveFiles(result.Resources), "html")
	if err != nil {
		return err
	}

	links, err := extractBrowseLinks(html, pageURL)
	if err != nil {
		return err
	}

	if links.IsProductGrid() {
		if err := tasks.CreateTask(domain.SourceDns, domain.PageTypeCategory.String(), pageURL); err != nil {
			return fmt.Errorf("save category %s: %w", pageURL, err)
		}
		stats.CategoriesCreated++
		logger.Info().
			Str("url", pageURL).
			Int("products_on_page", len(links.ListingURLs)).
			Int("pagination_pages", len(links.PaginationURLs)).
			Msg("dns bfs: category task created")

		for _, pagURL := range links.PaginationURLs {
			if !seen[pagURL] {
				*queue = append(*queue, pagURL)
				stats.InMemoryEnqueued++
				logger.Debug().Str("url", pagURL).Msg("dns bfs: pagination enqueued in memory")
			}
		}
		return nil
	}

	stats.HubsVisited++
	logger.Info().
		Str("url", pageURL).
		Int("subcategories", len(links.DiscoveryURLs)).
		Msg("dns bfs: hub visited (in memory only)")

	for _, hubURL := range links.DiscoveryURLs {
		if seen[hubURL] {
			continue
		}
		*queue = append(*queue, hubURL)
		stats.InMemoryEnqueued++
		logger.Debug().
			Str("from", pageURL).
			Str("hub", hubURL).
			Msg("dns bfs: subcategory enqueued in memory")
	}
	return nil
}

func toArchiveFiles(resources []domain.Resource) []*parser.ArchiveFile {
	files := make([]*parser.ArchiveFile, 0, len(resources))
	for _, res := range resources {
		files = append(files, &parser.ArchiveFile{Name: res.Name, Data: res.ResponseBody})
	}
	return files
}

func throttle(rateLimitRPS float64) {
	if rateLimitRPS <= 0 {
		return
	}
	time.Sleep(time.Duration(float64(time.Second) / rateLimitRPS))
}
