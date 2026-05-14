package yandex

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

type CompatibilityParser struct {
	brandAliases map[string]string
}

func NewCompatibilityParser(brandAliases map[string]string) *CompatibilityParser {
	return &CompatibilityParser{
		brandAliases: brandAliases,
	}
}

func (p *CompatibilityParser) Source() string {
	return domain.SourceYandex
}

func (p *CompatibilityParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) ([]*domain.DirectCompatibilityRecord, error) {
	if len(files) != 1 {
		return nil, fmt.Errorf("expected exactly 1 file in snapshot %d, got %d", pageSnapshotID, len(files))
	}
	htmlData := files[0].Data

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlData)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var records []*domain.DirectCompatibilityRecord

	doc.Find(".catalog .card").Each(func(i int, card *goquery.Selection) {
		content := card.Find(".card-content p:first-child")
		if content.Length() == 0 {
			return
		}
		strong := content.Find("strong")
		if strong.Length() == 0 {
			return
		}
		brandText := strings.TrimSpace(strong.Text())

		fullText := content.Text()
		idx := strings.Index(fullText, "|")
		var model string
		if idx != -1 {
			model = strings.TrimSpace(fullText[idx+1:])
		} else {
			model = strings.TrimSpace(strings.TrimPrefix(fullText, strong.Text()))
			model = strings.TrimPrefix(model, "|")
			model = strings.TrimSpace(model)
		}
		if model == "" {
			return
		}

		records = append(records, &domain.DirectCompatibilityRecord{
			PageSnapshotID: pageSnapshotID,
			Brand:          parser.NormalizeBrand(brandText, p.brandAliases),
			Model:          normalizeModel(model),
			Ecosystem:      "yandex",
			Protocol:       "zigbee",
		})
	})

	if len(records) == 0 {
		return nil, fmt.Errorf("no compatibility records found on page")
	}
	return records, nil
}

func (p *CompatibilityParser) normalizeBrand(brand string) string {
	if brand == "" {
		return ""
	}
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(brand)), " ", "-")
	if alias, ok := p.brandAliases[normalized]; ok {
		return alias
	}
	return normalized
}

func normalizeModel(model string) string {
	model = strings.ReplaceAll(model, "‑", "-")
	return strings.TrimSpace(model)
}
