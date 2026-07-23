package dns

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	ExtractorVersion    = "dns_listing_v2"
	fileHTML            = "html"
	fileCharacteristics = "characteristics.html"
	fileProductBuy      = "product-buy.json"
)

var (
	reviewCountRE      = regexp.MustCompile(`([0-5]\.\d+)\s*<span>\s*(\d+)\s+отзыв`)
	productImagesRE    = regexp.MustCompile(`initProductImagesSlider\([^,]+,\s*(\{.*?\})\);`)
	productModelCodeRE = regexp.MustCompile(`product-card-top__code-prefix">Код товара:\s*</span>\s*(\d+)`)
)

type ListingParser struct {
	brandAliases     map[string]string
	smartHomeMarkers []string
}

func NewListingParser(brandAliases map[string]string, smartHomeMarkers []string) *ListingParser {
	return &ListingParser{
		brandAliases:     brandAliases,
		smartHomeMarkers: smartHomeMarkers,
	}
}

func (p *ListingParser) Source() string { return domain.SourceDns }

func (p *ListingParser) Parse(pageSnapshotID int, files []*parser.ArchiveFile) (*domain.ListingParseResult, error) {
	htmlData, err := parser.FindFile(files, fileHTML)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", domain.SourceDns, err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlData)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	name := strings.TrimSpace(doc.Find("h1").First().Text())
	if name == "" {
		return nil, fmt.Errorf("product title not found")
	}

	specs := extractProductSpecs(doc)
	description := ""
	if charsData, err := parser.FindFile(files, fileCharacteristics); err == nil {
		if charsDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(charsData))); err == nil {
			mergeSpecs(specs, extractCharacteristicsSpecs(charsDoc))
			description = extractProductDescription(charsDoc)
		}
	}

	text := buildListingText(description, specs)
	brand := extractBrand(doc, specs)
	brand = parser.NormalizeBrand(brand, p.brandAliases)
	category := extractCategory(specs, doc)
	imageURL := extractProductImage(string(htmlData))
	rating, reviewCount := extractReviewCount(string(htmlData))

	if !p.containsSmartHomeMarker(name, text) {
		return &domain.ListingParseResult{PageSnapshotID: pageSnapshotID}, nil
	}

	res := &domain.ListingParseResult{
		PageSnapshotID:      pageSnapshotID,
		HasSmartHomeMarkers: true,
		ParsedAt:            time.Now(),
		ExtractorVer:        ExtractorVersion,
		Name:                name,
		Brand:               brand,
		InStock:             true,
		Text:                text,
		ImageURL:            imageURL,
		ReviewCount:         reviewCount,
		Rating:              rating,
	}

	if category != "" {
		res.Category = &category
	}
	if model := extractModelNumber(specs, string(htmlData)); model != "" {
		res.ModelNumber = &model
	}

	if buyData, err := parser.FindFile(files, fileProductBuy); err == nil {
		if buy, err := parseProductBuyJSON(buyData); err == nil {
			if buy.Name != "" {
				res.Name = buy.Name
			}
			if buy.Price > 0 {
				res.Price = &buy.Price
				currency := "RUB"
				res.Currency = &currency
				res.InStock = true
			}
		}
	}

	res.ContentHash = computeListingHash(res)
	return res, nil
}

type productBuyState struct {
	Name  string
	Price int
}

func parseProductBuyJSON(raw []byte) (productBuyState, error) {
	var resp struct {
		Result bool `json:"result"`
		Data   struct {
			States []struct {
				Data struct {
					Name  string `json:"name"`
					Price struct {
						Current int `json:"current"`
					} `json:"price"`
				} `json:"data"`
			} `json:"states"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return productBuyState{}, err
	}
	if !resp.Result || len(resp.Data.States) == 0 {
		return productBuyState{}, fmt.Errorf("empty product-buy response")
	}
	state := resp.Data.States[0].Data
	return productBuyState{
		Name:  strings.TrimSpace(state.Name),
		Price: state.Price.Current,
	}, nil
}

func extractProductSpecs(doc *goquery.Document) map[string]string {
	fields := make(map[string]string)
	doc.Find(".product-card-top__specs-item").Each(func(_ int, item *goquery.Selection) {
		label := strings.TrimSpace(item.Find(".product-card-top__specs-item-title").Text())
		value := strings.TrimSpace(item.Find(".product-card-top__specs-item-content").AttrOr("title", ""))
		if value == "" {
			value = strings.TrimSpace(item.Find(".product-card-top__specs-item-content").Text())
		}
		label = strings.TrimRight(label, ":")
		if label != "" && value != "" {
			fields[label] = value
		}
	})
	return fields
}

func extractCharacteristicsSpecs(doc *goquery.Document) map[string]string {
	fields := make(map[string]string)
	doc.Find(".product-characteristics__spec").Each(func(_ int, item *goquery.Selection) {
		label := strings.TrimSpace(item.Find(".product-characteristics__spec-title-content").Text())
		value := strings.TrimSpace(item.Find(".product-characteristics__spec-value").Text())
		label = strings.TrimRight(label, ":")
		if label != "" && value != "" {
			fields[label] = value
		}
	})
	return fields
}

func extractProductDescription(doc *goquery.Document) string {
	var parts []string
	doc.Find(".product-card-description-text").Each(func(_ int, block *goquery.Selection) {
		text := strings.TrimSpace(block.Text())
		if text != "" {
			parts = append(parts, text)
		}
	})
	return strings.Join(parts, "\n\n")
}

func mergeSpecs(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func buildListingText(description string, fields map[string]string) string {
	var sb strings.Builder
	if description != "" {
		sb.WriteString(description)
		sb.WriteString("\n\n")
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		fmt.Fprintf(&sb, "%s: %s\n", k, fields[k])
	}
	return strings.TrimSpace(sb.String())
}

func extractBrand(doc *goquery.Document, specs map[string]string) string {
	if alt := strings.TrimSpace(doc.Find(".product-card-top__brand-image").AttrOr("alt", "")); alt != "" {
		return alt
	}
	if brand := specs["Производитель"]; brand != "" {
		return brand
	}
	if brand := specs["Экосистема производителя"]; brand != "" {
		return brand
	}
	return specs["Приложение для управления"]
}

func extractCategory(specs map[string]string, doc *goquery.Document) string {
	if productType := specs["Тип"]; productType != "" {
		return productType
	}

	var crumbs []string
	doc.Find(".product-card__breadcrumbs .breadcrumb-list__item span[itemprop='name']").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" && text != "Каталог" {
			crumbs = append(crumbs, text)
		}
	})
	if len(crumbs) == 0 {
		return ""
	}
	return crumbs[len(crumbs)-1]
}

func extractProductImage(html string) string {
	match := productImagesRE.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	var payload struct {
		Images []struct {
			Desktop string `json:"desktop"`
		} `json:"images"`
	}
	if err := json.Unmarshal([]byte(match[1]), &payload); err != nil {
		return ""
	}
	if len(payload.Images) == 0 {
		return ""
	}
	return payload.Images[0].Desktop
}

func extractReviewCount(html string) (float64, int) {
	match := reviewCountRE.FindStringSubmatch(html)
	if len(match) < 3 {
		return 0, 0
	}
	count, _ := strconv.Atoi(match[2])
	review, _ := strconv.ParseFloat(match[1], 64)
	return review, count
}

func extractModelNumber(specs map[string]string, html string) string {
	if model := specs["Модель"]; model != "" {
		return model
	}
	return extractProductCode(html)
}

func extractProductCode(html string) string {
	match := productModelCodeRE.FindSubmatch([]byte(html))
	if len(match) < 2 {
		return ""
	}
	return string(match[1])
}

func (p *ListingParser) containsSmartHomeMarker(name, text string) bool {
	if len(p.smartHomeMarkers) == 0 {
		return true
	}
	joined := strings.ToLower(name + " " + text)
	for _, marker := range p.smartHomeMarkers {
		if strings.Contains(joined, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func computeListingHash(result *domain.ListingParseResult) string {
	model := ""
	if result.ModelNumber != nil {
		model = *result.ModelNumber
	}
	category := ""
	if result.Category != nil {
		category = *result.Category
	}
	price := 0
	if result.Price != nil {
		price = *result.Price
	}
	data := fmt.Sprintf("%s|%s|%s|%s|%d|%v|%s",
		result.Name, result.Brand, model, category, price, result.InStock, result.Text)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
