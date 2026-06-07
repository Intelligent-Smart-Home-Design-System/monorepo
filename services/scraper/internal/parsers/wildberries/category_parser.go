package wildberries

import (
    "fmt"
    "regexp"

    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

type CategoryParser struct{}

func NewCategoryParser() *CategoryParser {
    return &CategoryParser{}
}

func (p *CategoryParser) Source() string { return Source }

func (p *CategoryParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) ([]string, error) {
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

    re := regexp.MustCompile(`/catalog/(\d+)/detail\.aspx`)
    matches := re.FindAllStringSubmatch(string(htmlData), -1)
    if len(matches) == 0 {
        return nil, fmt.Errorf("no product links found in category page")
    }

    seen := make(map[string]bool)
    var productURLs []string
    for _, m := range matches {
        if len(m) > 1 {
            url := fmt.Sprintf("https://www.wildberries.ru/catalog/%s/detail.aspx", m[1])
            if !seen[url] {
                seen[url] = true
                productURLs = append(productURLs, url)
            }
        }
    }
    return productURLs, nil
}