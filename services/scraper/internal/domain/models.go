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
)

var pageTypes = map[string]PageType{
	"unknown":       PageTypeUnknown,
	"listing":       PageTypeListing,
	"discovery":     PageTypeDiscovery,
	"compatibility": PageTypeCompatibility,
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
	ID       int
	Source   string
	PageType PageType
	URL      string
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
	ScrapedAt     time.Time
	WARCBundle    []byte
	PageType      string
	SourceName    string
}

type ListingParseResult struct {
	PageSnapshotID int

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
	Processed    bool      `db:"processed"`
}
