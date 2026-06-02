package dns

import (
    "fmt"
    "regexp"
    "strings"

    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

type DiscoveryParser struct{}

func NewDiscoveryParser() *DiscoveryParser {
    return &DiscoveryParser{}
}

func (p *DiscoveryParser) Source() string { return domain.SourceDns }

func (p *DiscoveryParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) ([]string, error) {
    var htmlData []byte
    for _, f := range files {
        if f.Name == "html" {
            htmlData = f.Data
            break
        }
    }
    if len(htmlData) == 0 {
        return nil, fmt.Errorf("no HTML file found in snapshot %d", pageSnapshotID)
    }
    re := regexp.MustCompile(`/catalog/[^"']+/[^"']+`)
    matches := re.FindAllString(string(htmlData), -1)
    if len(matches) == 0 {
        return nil, fmt.Errorf("no product links found in search page")
    }

    seen := make(map[string]bool)
    var productURLs []string
    for _, m := range matches {
        if strings.Contains(m, "/catalog/") && !strings.Contains(m, "?") {
            fullURL := "https://www.dns-shop.ru" + m
            if !seen[fullURL] {
                seen[fullURL] = true
                productURLs = append(productURLs, fullURL)
            }
        }
    }
    if len(productURLs) == 0 {
        return nil, fmt.Errorf("no product URLs extracted")
    }
    return productURLs, nil
}