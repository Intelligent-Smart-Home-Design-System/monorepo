package wildberries

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

const (
	Source           = "wildberries"
	ExtractorVersion = "wb_listing_v1"

	fileDetail = "detail.json"
	fileCard   = "card.json"
)

type detailResponse struct {
	Products []detailProduct `json:"products"`
}

type detailProduct struct {
	ID           int          `json:"id"`
	Brand        string       `json:"brand"`
	Name         string       `json:"name"`
	Rating       float64      `json:"rating"`
	ReviewRating float64      `json:"reviewRating"`
	Feedbacks    int          `json:"feedbacks"`
	Sizes        []detailSize `json:"sizes"`
}

type detailSize struct {
	Price  detailPrice   `json:"price"`
	Stocks []detailStock `json:"stocks"`
}

type detailPrice struct {
	Basic   int `json:"basic"`
	Product int `json:"product"`
}

type detailStock struct {
	Qty int `json:"qty"`
}

type cardResponse struct {
	NmID        int          `json:"nm_id"`
	ImtName     string       `json:"imt_name"`
	VendorCode  string       `json:"vendor_code"`
	Description string       `json:"description"`
	SubjName    string       `json:"subj_name"`
	Options     []cardOption `json:"options"`
	Contents    string       `json:"contents"`
	Selling     cardSelling  `json:"selling"`
}

type cardOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type cardSelling struct {
	BrandName string `json:"brand_name"`
}

type ListingParser struct {
}

func NewListingParser() *ListingParser {
	return &ListingParser{}
}

func (p *ListingParser) Source() string { return Source }

// Parse extracts a ListingParseResult
func (p *ListingParser) Parse(pageSnapshotId int, files []*parser.ArchiveFile) (*domain.ListingParseResult, error) {
	detailData, err := parser.FindFile(files, fileDetail)
	if err != nil {
		return nil, fmt.Errorf("wildberries listing parser: %w", err)
	}
	cardData, err := parser.FindFile(files, fileCard)
	if err != nil {
		return nil, fmt.Errorf("wildberries listing parser: %w", err)
	}

	var detail detailResponse
	if err := json.Unmarshal(detailData, &detail); err != nil {
		return nil, fmt.Errorf("unmarshal detail.json: %w", err)
	}
	if len(detail.Products) == 0 {
		return nil, fmt.Errorf("detail.json contains no products")
	}
	prod := detail.Products[0]

	var card cardResponse
	if err := json.Unmarshal(cardData, &card); err != nil {
		return nil, fmt.Errorf("unmarshal card.json: %w", err)
	}

	res := &domain.ListingParseResult{
		PageSnapshotID: pageSnapshotId,
		ParsedAt:       time.Now(),
		ExtractorVer:   ExtractorVersion,
	}

	res.Name = strings.TrimSpace(prod.Name)

	// Prefer card selling brand; fall back to detail brand.
	res.Brand = normalizeBrand(card.Selling.BrandName)
	if res.Brand == "" {
		res.Brand = normalizeBrand(prod.Brand)
	}

	if m := findOption(card.Options, "Модель"); m != "" {
		res.ModelNumber = strPtr(m)
	}

	if card.SubjName != "" {
		res.Category = strPtr(card.SubjName)
	}

	totalQty, productPriceKopecks := stockAndPrice(prod.Sizes)
	res.InStock = totalQty > 0

	if productPriceKopecks > 0 {
		rubles := productPriceKopecks / 100
		res.Price = &rubles
		res.Currency = strPtr("RUB")
	}

	res.ImageURL = buildImageURL(prod.ID)
	res.Rating = prod.ReviewRating
	res.ReviewCount = prod.Feedbacks

	qty, qtyRaw := extractQuantity(card.Contents, card.Options)
	if qty > 0 {
		res.Quantity = &qty
	}
	if qtyRaw != "" {
		res.QuantityRaw = strPtr(qtyRaw)
	}

	res.Text = buildText(&card)

	return res, nil
}

func normalizeBrand(brand string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(brand)), " ", "-")
}

func stockAndPrice(sizes []detailSize) (totalQty, firstProductPrice int) {
	for _, sz := range sizes {
		for _, st := range sz.Stocks {
			totalQty += st.Qty
		}
		if sz.Price.Product > 0 && firstProductPrice == 0 {
			firstProductPrice = sz.Price.Product
		}
	}
	return
}

func findOption(opts []cardOption, name string) string {
	for _, o := range opts {
		if strings.EqualFold(strings.TrimSpace(o.Name), name) {
			return strings.TrimSpace(o.Value)
		}
	}
	return ""
}

func strPtr(s string) *string { return &s }

func buildImageURL(nmID int) string {
	vol := nmID / 100_000
	part := nmID / 1_000
	basket := 1
	return fmt.Sprintf(
		"https://mow-basket-cdn-%02d.geobasket.ru/vol%d/part%d/%d/images/big/1.webp",
		basket, vol, part, nmID,
	)
}

func extractQuantity(contents string, opts []cardOption) (int, string) {
	raw := findOption(opts, "Комплектация")
	if raw == "" {
		raw = strings.TrimSpace(contents)
	}
	if raw == "" {
		return 0, ""
	}
	return 1, raw
}

// buildText assembles the full description
func buildText(card *cardResponse) string {
	var sb strings.Builder
	if card.ImtName != "" {
		sb.WriteString(card.ImtName)
		sb.WriteString("\n\n")
	}
	if card.Description != "" {
		sb.WriteString(card.Description)
		sb.WriteString("\n\n")
	}
	for _, opt := range card.Options {
		sb.WriteString(opt.Name)
		sb.WriteString(": ")
		sb.WriteString(opt.Value)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}
