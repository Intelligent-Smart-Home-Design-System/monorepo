package wildberries

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

type DiscoveryParser struct{}

func NewDiscoveryParser() *DiscoveryParser {
	return &DiscoveryParser{}
}

func (p *DiscoveryParser) Source() string {
	return domain.SourceWildberries
}

func (p *DiscoveryParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) ([]string, error) {
	var productURLs []string
	seen := make(map[string]bool)

	for _, file := range files {
		if !strings.HasPrefix(file.Name, "page_") || !strings.HasSuffix(file.Name, ".json") {
			continue
		}
		var resp struct {
			Products []struct {
				ID int `json:"id"`
			} `json:"products"`
		}
		if err := json.Unmarshal(file.Data, &resp); err != nil {
			continue
		}
		for _, prod := range resp.Products {
			url := fmt.Sprintf("https://www.wildberries.ru/catalog/%d/detail.aspx", prod.ID)
			if !seen[url] {
				seen[url] = true
				productURLs = append(productURLs, url)
			}
		}
	}
	if len(productURLs) == 0 {
		return nil, fmt.Errorf("no product URLs found in discovery snapshot %d", pageSnapshotID)
	}
	return productURLs, nil
}
