package example

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

// DiscoveryParser разбирает snapshot страницы discovery (page_type=discovery).
//
// Подключение (паттерн Wildberries):
//   - parse.go → parser.NewWorker(logger, domain.PageTypeDiscovery, snapshotRepo,
//     []parser.SourceParser[[]string]{example.NewDiscoveryParser()})
//   - После Parse: для каждого URL → taskRepo.CreateTask(source, "listing", url)
//
// Если discovery у вас только через BFS (DNS), этот парсер может не понадобиться —
// оставьте заглушку или удалите регистрацию в parse.go.
type DiscoveryParser struct{}

// NewDiscoveryParser вызывается из internal/cli/parse.go.
func NewDiscoveryParser() *DiscoveryParser {
	return &DiscoveryParser{}
}

// Source возвращает имя источника для маршрутизации в parser.Worker.
// Должно совпадать с tracked_pages.source_name и domain.SourceExample.
func (p *DiscoveryParser) Source() string {
	panic("not implemented")
}

// Parse извлекает URL следующего уровня из tar.gz snapshot (parser.ExtractArchive).
//
// Вход:
//   - pageSnapshotID — page_snapshots.id (для сообщений об ошибках)
//   - files — распакованный архив; обычно parser.FindFile(files, "html") или *.json
//
// Выход:
//   - []string — абсолютные URL listing-страниц (или category, если ваш flow другой)
//
// Реализация: см. internal/parsers/wildberries/discovery_parser.go (JSON API)
// или internal/parsers/dns/browse_parser.go (HTML → BrowseLinks).
func (p *DiscoveryParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) ([]string, error) {
	panic("not implemented")
}
