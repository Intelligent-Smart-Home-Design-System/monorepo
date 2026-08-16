// Package example — шаблон HTTP-скрейпера. См. также internal/parsers/example/doc.go.
package example

import (
	"context"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

// Source — имя источника в tracked_pages.source_name и в CLI --sources.
// При копировании шаблона замените на domain.SourceYourShop.
const Source = domain.SourceExample

// Scraper загружает страницы и формирует domain.ScrapeResult для worker.
// Реализует scraper.Scraper (internal/scraper/worker.go).
type Scraper struct {
	timeout   time.Duration
	proxyURL  string
	userAgent string
	// exampleCfg config.ExampleConfig — раскомментируйте при реализации
}

// NewScraper создаёт скрейпер. Вызывается из internal/cli/scrape.go при регистрации source.
func NewScraper(scraping config.ScrapingConfig, exampleCfg config.ExampleConfig) *Scraper {
	panic("not implemented")
}

// Scrape выполняет HTTP-запрос(ы) для одной задачи из tracked_pages.
//
// Вход:
//   - task.ID — tracked_pages.id (worker подставит в ScrapeResult.TrackedPageID)
//   - task.URL — URL страницы или виртуальный (example://discovery/query для Pattern B)
//   - task.PageType — listing | discovery | category | compatibility
//
// По PageType (см. wildberries.Scrape switch):
//   - listing: GET html (+ опционально JSON API)
//   - discovery: Pattern A — GET task.URL; Pattern B — ParseVirtualDiscoveryURL + N×GET по шаблону
//   - category: GET html
//
// Выход:
//   - []domain.Resource с устойчивыми Name ("html", "detail.json", "page-1.json", …)
//
// Паттерны:
//   - Простая страница (sprut): один GET → "html"
//   - Listing + price API (dns): "html" + "product-buy.json"
//   - Discovery API (wildberries): page-1.json … из discovery_url_template
func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	panic("not implemented")
}
