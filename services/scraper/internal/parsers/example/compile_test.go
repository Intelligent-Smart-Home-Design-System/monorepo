package example

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
	exampleScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/example"
)

// Проверка на этапе компиляции: шаблон реализует нужные интерфейсы.
// Методы с panic намеренно не вызываются.
func TestExampleImplementsInterfaces(t *testing.T) {
	var _ scraper.Scraper = (*exampleScraper.Scraper)(nil)
	var _ parser.SourceParser[[]string] = (*DiscoveryParser)(nil)
	var _ parser.SourceParser[[]string] = (*CategoryParser)(nil)
	var _ parser.SourceParser[*domain.ListingParseResult] = (*ListingParser)(nil)
}
