package sprut

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

const (
	Source           = domain.SourceSprut
	ExtractorVersion = "sprut_listing_v1"
	fileHTML         = "html"
)

var ownersCountRe = regexp.MustCompile(`(\d+)`)

type ListingParser struct {
	brandAliases map[string]string
}

func NewListingParser(brandAliases map[string]string) *ListingParser {
	return &ListingParser{brandAliases: brandAliases}
}

func (p *ListingParser) Source() string { return Source }

func (p *ListingParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) (*domain.ListingParseResult, error) {
	htmlData, err := parser.FindFile(files, fileHTML)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", Source, err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlData)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	name := strings.TrimSpace(doc.Find("h1.header-title").First().Text())
	if name == "" {
		return nil, fmt.Errorf("product title not found")
	}

	brand, deviceType := parsePretitle(doc.Find(".header-pretitle").First().Text())
	brand = parser.NormalizeBrand(brand, p.brandAliases)

	fields := extractLabeledFields(doc)

	category := fields["Тип устройства"]
	if category == "" {
		category = deviceType
	}
	model := fields["Модель"]

	res := &domain.ListingParseResult{
		PageSnapshotID:      pageSnapshotID,
		HasSmartHomeMarkers: true,
		ParsedAt:            time.Now(),
		ExtractorVer:        ExtractorVersion,
		Name:                name,
		Brand:               brand,
		InStock:             true,
		Text:                buildCharacteristicsText(fields),
		Rating:              extractRating(doc),
		ReviewCount:         extractReviewCount(doc),
	}

	if model != "" {
		res.ModelNumber = &model
	}
	if category != "" {
		res.Category = &category
	}
	if imgURL := doc.Find(".image img[src]").First().AttrOr("src", ""); imgURL != "" {
		res.ImageURL = imgURL
	}

	res.ContentHash = computeHash(res)
	return res, nil
}

func parsePretitle(pretitle string) (brand, deviceType string) {
	pretitle = strings.TrimSpace(pretitle)
	if pretitle == "" {
		return "", ""
	}
	parts := strings.Split(pretitle, "•")
	if len(parts) >= 1 {
		brand = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		deviceType = strings.TrimSpace(parts[1])
	}
	return brand, deviceType
}

func extractLabeledFields(doc *goquery.Document) map[string]string {
	fields := make(map[string]string)

	doc.Find("h3").Each(func(_ int, heading *goquery.Selection) {
		section := strings.TrimSpace(heading.Text())
		if section != "Данные" && section != "Характеристики" && section != "Служебные" {
			return
		}

		info := heading.NextFiltered(".info")
		if info.Length() == 0 {
			return
		}

		info.Find("div").Each(func(_ int, row *goquery.Selection) {
			label := strings.TrimSpace(strings.TrimSuffix(row.Find("b").First().Text(), ":"))
			if label == "" {
				return
			}
			value := extractFieldValue(row)
			if value == "" {
				return
			}
			key := label
			if section != "Данные" {
				key = section + ": " + label
			}
			fields[key] = value
		})
	})

	return fields
}

func extractFieldValue(row *goquery.Selection) string {
	rowClone := row.Clone()
	rowClone.Find("b").Remove()

	var parts []string
	rowClone.Find("a").Each(func(_ int, link *goquery.Selection) {
		if text := strings.TrimSpace(link.Text()); text != "" {
			parts = append(parts, text)
		}
	})
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}

	return strings.TrimSpace(rowClone.Text())
}

func buildCharacteristicsText(fields map[string]string) string {
	skipLabels := map[string]bool{
		"Модель":          true,
		"Тип устройства":  true,
	}

	var sb strings.Builder
	keys := make([]string, 0, len(fields))
	for k := range fields {
		label := k
		if idx := strings.Index(k, ": "); idx != -1 {
			label = k[idx+2:]
		}
		if skipLabels[label] {
			continue
		}
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		fmt.Fprintf(&sb, "%s: %s\n", k, fields[k])
	}
	return strings.TrimSpace(sb.String())
}

func extractRating(doc *goquery.Document) float64 {
	filled := 0
	doc.Find("span.vue-star-rating-star").Each(func(_ int, star *goquery.Selection) {
		content := star.Text() + star.Find("svg").Text()
		if html, err := star.Html(); err == nil {
			content += html
		}
		if strings.Contains(content, `stop-color="#f6c343"`) {
			filled++
		}
	})
	if filled == 0 {
		return 0
	}
	return float64(filled)
}

func extractReviewCount(doc *goquery.Document) int {
	reviewsLink := doc.Find(`a[href*="#reviews"]`).First()
	text := strings.TrimSpace(reviewsLink.Text())
	if text == "" {
		return 0
	}
	match := ownersCountRe.FindString(text)
	if match == "" {
		return 0
	}
	count, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	return count
}

func computeHash(result *domain.ListingParseResult) string {
	model := ""
	if result.ModelNumber != nil {
		model = *result.ModelNumber
	}
	category := ""
	if result.Category != nil {
		category = *result.Category
	}
	data := fmt.Sprintf("%s|%s|%s|%s|%s", result.Name, result.Brand, model, category, result.Text)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
