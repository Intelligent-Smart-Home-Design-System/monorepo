package example

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

// CategoryParser разбирает snapshot category-страницы (page_type=category).
//
// Два варианта подключения в parse.go:
//
//  A) Простой (Wildberries): SourceParser[[]string] + generic parser.Worker
//     → CreateTask(listing) для каждого URL из Parse.
//
//  B) С пагинацией (DNS): Parse возвращает *BrowseLinks, обработка в
//     ParseCategorySnapshots (см. example/integration.go и cli/parse.go parseDNSCategorySnapshots).
//
// Запускается после scrape --discovery (category snapshots) или scrape --page-types category.
type CategoryParser struct{}

// NewCategoryParser вызывается из internal/cli/parse.go.
func NewCategoryParser() *CategoryParser {
	return &CategoryParser{}
}

// Source возвращает имя источника для parser.Worker или ручной маршрутизации.
func (p *CategoryParser) Source() string {
	panic("not implemented")
}

// Parse извлекает ссылки с category/grid страницы.
//
// Если используете паттерн B (DNS), измените сигнатуру на (*BrowseLinks, error)
// и не регистрируйте в generic Worker — вызывайте ParseCategorySnapshots.
//
// Вход: pageSnapshotID, files (как у DiscoveryParser).
// Выход: []string listing URL — по одному на товар на текущей странице сетки.
func (p *CategoryParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) ([]string, error) {
	panic("not implemented")
}
