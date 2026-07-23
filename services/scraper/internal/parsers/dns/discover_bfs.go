package dns

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
)

// DiscoverStats summarizes in-memory DNS catalog BFS during scrape --discovery.
type DiscoverStats struct {
	HubsVisited       int
	CategoriesCreated int
	InMemoryEnqueued  int
	Fetches           int
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
	scraperInst scraper.Scraper,
) (DiscoverStats, error) {
	stats := DiscoverStats{}

	queue := append([]string{}, seeds...)
	seen := make(map[string]bool, len(seeds))

	for len(queue) > 0 {
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}
		if dnsCfg.MaxBFSFetches > 0 && stats.Fetches >= dnsCfg.MaxBFSFetches {
			logger.Info().Int("max_bfs_fetches", dnsCfg.MaxBFSFetches).Msg("dns bfs: fetch limit reached")
			break
		}
		url := queue[0]
		queue = queue[1:]
		if seen[url] {
			continue
		}
		seen[url] = true

		if err := processPage(ctx, logger, scraperInst, scraping.RateLimitRps, url, tasks, &stats, seen, &queue); err != nil {
			logger.Warn().Err(err).Str("url", url).Msg("dns bfs: page failed")
		} else {
			stats.Fetches++
		}
	}

	logger.Info().
		Int("fetches", stats.Fetches).
		Int("hubs_visited", stats.HubsVisited).
		Int("categories_created", stats.CategoriesCreated).
		Int("in_memory_enqueued", stats.InMemoryEnqueued).
		Msg("dns bfs finished")

	return stats, nil
}

func processPage(
	ctx context.Context,
	logger zerolog.Logger,
	scraperInst scraper.Scraper,
	rateLimitRPS float64,
	pageURL string,
	tasks categoryTaskWriter,
	stats *DiscoverStats,
	seen map[string]bool,
	queue *[]string,
) error {
	throttle(rateLimitRPS)

	result, err := scraperInst.Scrape(ctx, domain.ScrapeTask{URL: pageURL})
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
		diag := DiagnoseBrowseHTML(html)
		evt := logger.Warn().Err(err).Str("url", pageURL)
		for k, v := range diag.LogFields() {
			evt = evt.Interface(k, v)
		}
		if diag.PageKind == "empty_shell" {
			evt.Msg("dns bfs: empty page shell — JS likely not rendered yet; check browser wait")
		} else {
			evt.Msg("dns bfs: no category or product links in HTML")
		}
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
