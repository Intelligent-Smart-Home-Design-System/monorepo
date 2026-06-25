package sprut

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
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
	ExtractorVersion = "sprut_listing_v2"
	fileHTML         = "html"
)

var digitsRe = regexp.MustCompile(`(\d+)`)

// Postgres column parsed_listing_snapshots.extracted_rating is NUMERIC(3,2).
const maxExtractedRating = 9.99

var sectionTitles = map[string]bool{
	"Данные":          true,
	"Характеристики":  true,
	"Служебные":       true,
}

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

	brand, deviceType := parsePretitle(doc)
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
		ImageURL:            extractImageURL(doc),
	}

	if model != "" {
		res.ModelNumber = &model
	}
	if category != "" {
		res.Category = &category
	}

	res.ContentHash = computeHash(res)
	return res, nil
}

func catalogItemRoot(doc *goquery.Document) *goquery.Selection {
	root := doc.Find(".catalog-item").First()
	if root.Length() == 0 {
		return doc.Selection
	}
	return root
}

func parsePretitle(doc *goquery.Document) (brand, deviceType string) {
	pretitle := doc.Find(".header-pretitle").First()
	links := pretitle.Find("a")
	if links.Length() >= 1 {
		brand = strings.TrimSpace(links.First().Text())
	}
	if links.Length() >= 2 {
		deviceType = strings.TrimSpace(links.Eq(1).Text())
	}
	if brand != "" {
		return brand, deviceType
	}

	text := strings.TrimSpace(pretitle.Text())
	if text == "" {
		return "", ""
	}
	parts := strings.Split(text, "•")
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
	root := catalogItemRoot(doc)

	section := ""
	root.Find("h3, .catalog-item-param").Each(func(_ int, el *goquery.Selection) {
		if el.Is("h3") {
			section = strings.TrimSpace(el.Text())
			return
		}
		if !sectionTitles[section] {
			return
		}

		label, value := parseParamRow(el)
		if label == "" || value == "" {
			return
		}

		key := label
		if section != "Данные" {
			key = section + ": " + label
		}
		fields[key] = value
	})

	// Legacy catalog-card layout (.info rows).
	if len(fields) == 0 {
		extractLegacyInfoFields(root, fields)
	}

	return fields
}

func normalizeLabel(raw string) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(raw), ":"))
}

func parseParamRow(row *goquery.Selection) (label, value string) {
	label = normalizeLabel(row.Find(".text-secondary").First().Text())
	if label == "" {
		label = normalizeLabel(row.Find("b").First().Text())
	}
	if label == "" {
		return "", ""
	}
	value = extractColValue(row.Find(".col").First())
	if value == "" {
		value = extractFieldValue(row)
	}
	return label, value
}

func extractColValue(col *goquery.Selection) string {
	if col.Length() == 0 {
		return ""
	}

	links := col.Find("a")
	if links.Length() > 0 {
		var parts []string
		links.Each(func(_ int, link *goquery.Selection) {
			if text := strings.TrimSpace(link.Text()); text != "" {
				parts = append(parts, text)
			}
		})
		return strings.Join(parts, ", ")
	}

	var parts []string
	col.Children().Each(func(_ int, node *goquery.Selection) {
		if text := strings.TrimSpace(node.Text()); text != "" {
			parts = append(parts, text)
		}
	})
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}

	return strings.TrimSpace(col.Text())
}

func extractLegacyInfoFields(root *goquery.Selection, fields map[string]string) {
	root.Find("h3").Each(func(_ int, heading *goquery.Selection) {
		section := strings.TrimSpace(heading.Text())
		if !sectionTitles[section] {
			return
		}

		info := heading.NextFiltered(".info")
		if info.Length() == 0 {
			return
		}

		info.Find("div").Each(func(_ int, row *goquery.Selection) {
			label := normalizeLabel(row.Find("b").First().Text())
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
		"Модель":         true,
		"Тип устройства": true,
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

func extractImageURL(doc *goquery.Document) string {
	selectors := []string{
		".catalog-item-header img.img-fluid",
		".catalog-item-header .avatar img",
		".catalog-item .image img",
	}
	for _, sel := range selectors {
		src := strings.TrimSpace(doc.Find(sel).First().AttrOr("src", ""))
		if src != "" && !strings.Contains(src, "placeholder") {
			return src
		}
	}
	return ""
}

func extractRating(doc *goquery.Document) float64 {
	scope := catalogItemRoot(doc).Find(".catalog-item-header .vue-star-rating").First()
	if scope.Length() == 0 {
		scope = catalogItemRoot(doc).Find(".catalog-item-row-stat .vue-star-rating").First()
	}
	if scope.Length() == 0 {
		scope = catalogItemRoot(doc).Find(".vue-star-rating").First()
	}
	if scope.Length() == 0 {
		return 0
	}

	filled := 0
	scope.Find("span.vue-star-rating-star").Each(func(_ int, star *goquery.Selection) {
		html, err := star.Html()
		if err != nil {
			return
		}
		if strings.Contains(html, `stop-color="#f6c343"`) {
			filled++
		}
	})
	if filled == 0 {
		return 0
	}
	return clampRating(float64(filled))
}

func clampRating(rating float64) float64 {
	if rating > 5 {
		rating = 5
	}
	if rating > maxExtractedRating {
		rating = maxExtractedRating
	}
	if rating < 0 {
		return 0
	}
	return math.Round(rating*100) / 100
}

func extractReviewCount(doc *goquery.Document) int {
	root := catalogItemRoot(doc)
	if count := root.Find(".reviews .comments > .comment.level-1").Length(); count > 0 {
		return count
	}

	reviewsLink := doc.Find(`a[href*="#reviews"]`).First()
	text := strings.TrimSpace(reviewsLink.Text())
	if text != "" {
		if match := digitsRe.FindString(text); match != "" {
			if count, err := strconv.Atoi(match); err == nil {
				return count
			}
		}
	}

	return 0
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
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s", result.Name, result.Brand, model, category, result.Text, result.ImageURL)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
