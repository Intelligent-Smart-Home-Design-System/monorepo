package example

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
)

// BootstrapDiscoverySeeds создаёт начальные discovery-задачи в tracked_pages из конфига.
//
// Вызывается из internal/cli/scrape.go при scrape --discovery и discovery_bootstrap содержит "seed".
//
// Pattern A — готовые URL (DNS):
//   - exampleCfg.DiscoverySeeds → CreateTask(source, "discovery", url)
//   - exampleCfg.SearchQueries + SearchURLTemplate → CreateTask("discovery", builtURL) для page 1..MaxPages
//
// Pattern B — текст + шаблон (WB):
//   - exampleCfg.DiscoveryTextQueries → CreateTask(source, "discovery", "example://discovery/"+query)
//   - Скрейпер в Scrape() разбирает виртуальный URL и подставляет query/page в DiscoveryURLTemplate
//
// Не создавайте listing здесь — только точки входа для discovery/BFS/API.
func BootstrapDiscoverySeeds(taskRepo *repository.TrackedPageRepo, exampleCfg config.ExampleConfig) error {
	panic("not implemented")
}

// BootstrapRegularTasks создаёт задачи вне пайплайна discovery (обычный scrape).
//
// Вызывается из internal/cli/scrape.go когда НЕ передан флаг --discovery (как WB category_url).
//
//   - exampleCfg.CategoryURLs → CreateTask(source, "category", url) для каждого
//   - exampleCfg.ListingURLs → CreateTask(source, "listing", url) для каждого
//
// После bootstrap: scrape --page-types category|listing → parse (без --discovery).
func BootstrapRegularTasks(taskRepo *repository.TrackedPageRepo, exampleCfg config.ExampleConfig) error {
	panic("not implemented")
}

// ParseCategorySnapshots — (опционально) category snapshots, DNS-стиль. См. doc.go про listingsOnly.
func ParseCategorySnapshots(
	ctx context.Context,
	logger zerolog.Logger,
	snapshotRepo *repository.SnapshotRepo,
	taskRepo *repository.TrackedPageRepo,
	jobs config.JobsConfig,
	job string,
	listingsOnly bool,
) int {
	panic("not implemented")
}

// BuildDiscoveryAPIURL подставляет {query} и {page} в discovery_url_template (Pattern B, как WB).
func BuildDiscoveryAPIURL(template, query string, page int) (string, error) {
	panic("not implemented")
}

// ParseVirtualDiscoveryURL извлекает текст запроса из example://discovery/{query} (Pattern B).
func ParseVirtualDiscoveryURL(taskURL string) (query string, err error) {
	panic("not implemented")
}
