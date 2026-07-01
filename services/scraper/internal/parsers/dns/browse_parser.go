package dns

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

// DiscoveryParser is used only for tests; production BFS is RunDiscoveryBFS.
type DiscoveryParser struct{}

func NewDiscoveryParser() *DiscoveryParser {
	return &DiscoveryParser{}
}

func (p *DiscoveryParser) Source() string { return domain.SourceDns }

func (p *DiscoveryParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) (*BrowseLinks, error) {
	htmlData, err := parser.FindFile(files, "html")
	if err != nil {
		return nil, fmt.Errorf("%s snapshot %d: %w", domain.SourceDns, pageSnapshotID, err)
	}
	return extractBrowseLinks(htmlData, "")
}

// CategoryParser extracts product listings and pagination from scraped category snapshots.
type CategoryParser struct{}

func NewCategoryParser() *CategoryParser {
	return &CategoryParser{}
}

func (p *CategoryParser) Source() string { return domain.SourceDns }

func (p *CategoryParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) (*BrowseLinks, error) {
	htmlData, err := parser.FindFile(files, "html")
	if err != nil {
		return nil, fmt.Errorf("%s snapshot %d: %w", domain.SourceDns, pageSnapshotID, err)
	}
	return extractCategoryLinks(htmlData, "")
}
