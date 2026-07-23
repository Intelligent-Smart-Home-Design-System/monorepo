package sprut

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

// DiscoverStats summarizes the in-memory sprut catalog walk during scrape --discovery.
type DiscoverStats struct {
	HubsVisited       int
	CategoriesCreated int
	ItemPagesCreated  int
	Fetches           int
}

type categoryTaskWriter interface {
	CreateTask(source, pageType, url string) error
}

// RunDiscoveryBFS walks hub pages (section → light → ...) as HTML, since those load fine.
// For every subcategory link found, it resolves the node via api.sprut.ai (catalog_id +
// item count) instead of fetching its HTML: leaf categories (real product-grid pages, e.g.
// /catalog/light/lightbulb) 500 on direct SSR load, so their items are pulled straight from
// /catalogs/items and persisted as page_type=category tasks pointing at the API URL — the
// human page is never fetched. Nodes with zero items are hubs and get queued for HTML walk.
func RunDiscoveryBFS(
	ctx context.Context,
	logger zerolog.Logger,
	scraping config.ScrapingConfig,
	sprutCfg config.SprutConfig,
	seeds []string,
	tasks categoryTaskWriter,
	scraperInst scraper.Scraper,
) (DiscoverStats, error) {
	stats := DiscoverStats{}

	queue := append([]string{}, seeds...)
	seen := make(map[string]bool, len(seeds))
	for _, seed := range seeds {
		seen[seed] = true
	}

	for len(queue) > 0 {
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}
		if sprutCfg.MaxBFSFetches > 0 && stats.Fetches >= sprutCfg.MaxBFSFetches {
			logger.Info().Int("max_bfs_fetches", sprutCfg.MaxBFSFetches).Msg("sprut bfs: fetch limit reached")
			break
		}
		hubURL := queue[0]
		queue = queue[1:]

		throttle(scraping.RateLimitRps)
		links, err := fetchHub(ctx, scraperInst, hubURL)
		if err != nil {
			logger.Warn().Err(err).Str("url", hubURL).Msg("sprut bfs: hub page failed")
			continue
		}
		stats.Fetches++
		stats.HubsVisited++
		logger.Info().Str("url", hubURL).Int("children", len(links.DiscoveryURLs)).Msg("sprut bfs: hub visited")

		for _, childURL := range links.DiscoveryURLs {
			if seen[childURL] {
				continue
			}
			seen[childURL] = true

			isHub, err := classifyAndEnqueue(ctx, logger, scraperInst, childURL, tasks, &stats)
			if err != nil {
				logger.Warn().Err(err).Str("url", childURL).Msg("sprut bfs: classify failed")
				continue
			}
			if isHub {
				queue = append(queue, childURL)
			}
		}
	}

	logger.Info().
		Int("fetches", stats.Fetches).
		Int("hubs_visited", stats.HubsVisited).
		Int("categories_created", stats.CategoriesCreated).
		Int("item_pages_created", stats.ItemPagesCreated).
		Msg("sprut bfs finished")

	return stats, nil
}

func fetchHub(ctx context.Context, scraperInst scraper.Scraper, hubURL string) (*BrowseLinks, error) {
	result, err := scraperInst.Scrape(ctx, domain.ScrapeTask{URL: hubURL})
	if err != nil {
		return nil, err
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("empty scrape result for %s", hubURL)
	}
	html, err := parser.FindFile(toArchiveFiles(result.Resources), "html")
	if err != nil {
		return nil, err
	}
	return extractBrowseLinks(html, hubURL)
}

// classifyAndEnqueue resolves catalogURL's slug against api.sprut.ai. If the node has
// items, every items-API page is persisted as a page_type=category task and it reports
// isHub=false (nothing more to walk). Otherwise it reports isHub=true so the caller queues
// catalogURL for an HTML fetch to discover its own children.
func classifyAndEnqueue(
	ctx context.Context,
	logger zerolog.Logger,
	scraperInst scraper.Scraper,
	catalogURL string,
	tasks categoryTaskWriter,
	stats *DiscoverStats,
) (isHub bool, err error) {
	slug := lastPathSegment(catalogURL)
	if slug == "" {
		return false, fmt.Errorf("cannot extract slug from %s", catalogURL)
	}

	catalogID, err := resolveCatalogID(ctx, scraperInst, slug)
	if err != nil {
		return false, fmt.Errorf("resolve catalog id for %s: %w", slug, err)
	}

	firstPage, err := fetchItemsPage(ctx, scraperInst, catalogID, 1)
	if err != nil {
		return false, fmt.Errorf("fetch items page 1 for catalog %d (%s): %w", catalogID, slug, err)
	}

	if firstPage.Meta.TotalCount == 0 {
		return true, nil
	}

	pageCount := firstPage.Meta.PageCount
	if pageCount < 1 {
		pageCount = 1
	}
	for page := 1; page <= pageCount; page++ {
		apiURL := catalogItemsURL(catalogID, page)
		if err := tasks.CreateTask(domain.SourceSprut, domain.PageTypeCategory.String(), apiURL); err != nil {
			return false, fmt.Errorf("save category page %d for catalog %d: %w", page, catalogID, err)
		}
		stats.ItemPagesCreated++
	}
	stats.CategoriesCreated++
	logger.Info().
		Str("url", catalogURL).
		Str("slug", slug).
		Int("catalog_id", catalogID).
		Int("total_items", firstPage.Meta.TotalCount).
		Int("pages", pageCount).
		Msg("sprut bfs: leaf category resolved via API")

	return false, nil
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
