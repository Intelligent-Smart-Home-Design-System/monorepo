package dns

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

// BrowseParser extracts child category and product URLs from DNS catalog/search pages.
type BrowseParser struct{}

func NewBrowseParser() *BrowseParser {
	return &BrowseParser{}
}

func (p *BrowseParser) Source() string { return domain.SourceDns }

func (p *BrowseParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) (*BrowseLinks, error) {
	htmlData, err := parser.FindFile(files, "html")
	if err != nil {
		return nil, fmt.Errorf("%s snapshot %d: %w", domain.SourceDns, pageSnapshotID, err)
	}
	return extractBrowseLinks(htmlData)
}

// DiscoveryParser is an alias for browse pages reached via search / bootstrap seeds.
type DiscoveryParser = BrowseParser

func NewDiscoveryParser() *BrowseParser {
	return NewBrowseParser()
}

// CategoryParser is an alias for browse pages reached via catalog BFS.
type CategoryParser = BrowseParser

func NewCategoryParser() *BrowseParser {
	return NewBrowseParser()
}
