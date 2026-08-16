package domain

import (
	"net/http"
	"time"
)

type PageType uint

const (
	PageTypeUnknown PageType = iota
	PageTypeListing
	PageTypeDiscovery
	PageTypeCompatibility
	PageTypeCategory
)

var pageTypes = map[string]PageType{
	"unknown":       PageTypeUnknown,
	"listing":       PageTypeListing,
	"discovery":     PageTypeDiscovery,
	"compatibility": PageTypeCompatibility,
	"category":      PageTypeCategory,
}

func (t PageType) String() string {
	for s, i := range pageTypes {
		if i == t {
			return s
		}
	}
	return PageTypeUnknown.String()
}

func PageTypeFromString(s string) PageType {
	t, ok := pageTypes[s]
	if !ok {
		return PageTypeUnknown
	}
	return t
}

type ScrapeTask struct {
	ID            int
	Source        string
	PageType      PageType
	URL           string
	FirstSeenAt   time.Time  // tracked_pages.first_seen_at — появление задачи в пайплайне
	LastScrapedAt *time.Time // tracked_pages.last_scraped_at; nil — ещё не скрапили
}

type ScrapeResult struct {
	Err           error
	TrackedPageID int
	DurationMs    int
	Resources     []Resource
}

// assuming GET method
type Resource struct {
	Name           string
	URL            string
	RequestHeaders http.Header

	StatusCode      int
	Status          string
	ResponseHeaders http.Header
	ResponseBody    []byte

	Timestamp time.Time // when it was fetched
}

type PageSnapshot struct {
	ID            int
	TrackedPageID int
	PageURL       string
	ScrapedAt     time.Time
	WARCBundle    []byte
	PageType      string
	SourceName    string
}

type ListingParseResult struct {
	PageSnapshotID int

	HasSmartHomeMarkers bool

	InStock      bool
	Text         string
	Name         string
	Brand        string
	ImageURL     string
	Price        *int
	Currency     *string
	ModelNumber  *string
	Category     *string
	Quantity     *int
	QuantityRaw  *string
	Rating       float64
	ReviewCount  int
	ContentHash  string
	ExtractorVer string
	ParsedAt     time.Time
	Processed    bool
}

const (
    SourceSprut       = "sprut"
    SourceWildberries = "wildberries"
    SourcePrinter     = "printer"
    SourceYandex      = "yandex"
	SourceDns         = "dns"
	SourceApifyYandexMarket = "apify_yandex_market"
	// SourceExample — учебный шаблон (internal/scrapers/example, internal/parsers/example).
	// Скопируйте пакеты и переименуйте перед продакшеном.
	SourceExample     = "example"
)

type DirectCompatibilityRecord struct {
    PageSnapshotID int
    Brand          string
    Model          string
    Ecosystem      string
    Protocol       string
}
