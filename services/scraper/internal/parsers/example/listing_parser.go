package example

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

const (
	// ExtractorVersion — версия экстрактора для parsed_listing_snapshots.extractor_ver.
	// Увеличивайте при изменении логики полей (sprut: "sprut_listing_v2", dns: "dns_listing_v1").
	ExtractorVersion = "example_listing_v0"
)

// ListingParser разбирает snapshot карточки товара (page_type=listing).
//
// Подключение в parse.go:
//   listingParsers = append(listingParsers, example.NewListingParser(cfg.Example.BrandAliases, markers))
//   listingWorker := parser.NewWorker(logger, domain.PageTypeListing, snapshotRepo, listingParsers)
//
// После Parse worker сохраняет результат через snapshotRepo.SaveListingParseResult,
// если listing.HasSmartHomeMarkers == true (фильтр в cli/parse.go).
type ListingParser struct {
	brandAliases          map[string]string
	smartHomeDeviceMarkers []string
}

// NewListingParser принимает настройки из config.Example или shared wildberries markers.
func NewListingParser(brandAliases map[string]string, smartHomeDeviceMarkers []string) *ListingParser {
	return &ListingParser{
		brandAliases:          brandAliases,
		smartHomeDeviceMarkers: smartHomeDeviceMarkers,
	}
}

// Source возвращает имя источника.
func (p *ListingParser) Source() string {
	panic("not implemented")
}

// Parse извлекает структурированные поля товара из snapshot.
//
// Вход:
//   - files — как минимум "html"; опционально JSON с ценой (как dns "product-buy.json")
//
// Выход — domain.ListingParseResult:
//   - PageSnapshotID — передайте pageSnapshotID из аргумента
//   - Name, Brand, Text, ImageURL, Price, Currency, InStock, Rating, ReviewCount
//   - HasSmartHomeMarkers — true если товар относится к умному дому (по маркерам в тексте/характеристиках)
//   - ContentHash — sha256 канонического текста (для дедупликации)
//   - ExtractorVer — ExtractorVersion
//   - ParsedAt — time.Now()
//
// См. internal/parsers/sprut/listing_parser.go и internal/parsers/dns/listing_parser.go.
func (p *ListingParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) (*domain.ListingParseResult, error) {
	panic("not implemented")
}
