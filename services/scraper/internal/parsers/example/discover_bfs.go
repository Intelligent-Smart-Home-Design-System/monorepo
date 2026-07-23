package example

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
)

// DiscoverStats — метрики обхода каталога (логируются после RunDiscoveryBFS).
type DiscoverStats struct {
	HubsVisited       int // hub-страниц обработано в памяти (не в БД)
	CategoriesCreated int // CreateTask(category) — попадут в tracked_pages
	InMemoryEnqueued  int // URL добавлено в очередь BFS, ещё не обработано
}

// categoryTaskWriter — минимальный интерфейс для CreateTask без импорта repository.
// *repository.TrackedPageRepo удовлетворяет этому интерфейсу.
type categoryTaskWriter interface {
	CreateTask(source, pageType, url string) error
}

// RunDiscoveryBFS обходит каталог от seeds и создаёт только category-задачи в БД.
//
// Когда нужен: каталог с hub-страницами (подкатегории), как DNS.
// Когда не нужен: фиксированный список category URL или API discovery (Wildberries).
//
// Подключение в internal/cli/scrape.go, ветка discoveryOnly:
//   if seedBootstrap && len(cfg.Example.DiscoverySeeds) > 0 {
//       example.RunDiscoveryBFS(ctx, logger, cfg.Scraping, cfg.Example, cfg.Example.DiscoverySeeds, taskRepo)
//   }
//
// Алгоритм (копируйте из internal/parsers/dns/discover_bfs.go):
//  1. Очередь URL ← seeds
//  2. Для каждого URL: Scraper.Scrape → ExtractBrowseLinks
//  3. Hub (не product grid): enqueue DiscoveryURLs в память, не писать в БД
//  4. Product grid: CreateTask(category, url); enqueue PaginationURLs в память
//  5. После BFS: scrape --discovery доскрапит category → snapshots → parse --discovery
//
// Snapshots в BFS не сохраняйте — worker сохранит их на фазе scrape category (WB/DNS стиль).
func RunDiscoveryBFS(
	ctx context.Context,
	logger zerolog.Logger,
	scraping config.ScrapingConfig,
	exampleCfg config.ExampleConfig,
	seeds []string,
	tasks categoryTaskWriter,
) (DiscoverStats, error) {
	panic("not implemented")
}
