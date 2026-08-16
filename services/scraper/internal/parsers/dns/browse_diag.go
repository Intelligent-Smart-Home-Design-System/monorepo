package dns

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// BrowseDiagnostics describes what extractBrowseLinks would see in HTML.
type BrowseDiagnostics struct {
	HTMLBytes              int
	Title                  string
	SubcategoryAnchorCount int
	ProductAnchorCount     int
	CatalogHrefCount       int
	ProductHrefCount       int
	HasSubcategoryBlock    bool
	HasProductsPage        bool
	HasHeaderPlug          bool
	PageKind               string // hub, grid, empty_shell, unknown
}

var titleRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func DiagnoseBrowseHTML(html []byte) BrowseDiagnostics {
	text := string(html)
	d := BrowseDiagnostics{
		HTMLBytes:       len(html),
		Title:           extractBrowseTitle(text),
		HasHeaderPlug:   strings.Contains(text, "header-plug"),
		HasSubcategoryBlock: strings.Contains(text, `data-subcategory-container`) ||
			strings.Contains(text, "subcategory__item"),
		HasProductsPage: strings.Contains(text, "products-page"),
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		d.PageKind = "unparseable"
		return d
	}

	d.SubcategoryAnchorCount = doc.Find("a.subcategory__item[href]").Length()
	d.ProductAnchorCount = doc.Find("a.catalog-product__image-link[href]").Length()

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href := s.AttrOr("href", "")
		if isProductPath(href) {
			d.ProductHrefCount++
		} else if isCategoryPath(href) {
			d.CatalogHrefCount++
		}
	})

	switch {
	case d.SubcategoryAnchorCount > 0:
		d.PageKind = "hub"
	case d.ProductAnchorCount > 0:
		d.PageKind = "grid"
	case d.HTMLBytes < 12_000 && !d.HasSubcategoryBlock && !d.HasProductsPage:
		d.PageKind = "empty_shell"
	default:
		d.PageKind = "unknown"
	}
	return d
}

func extractBrowseTitle(html string) string {
	m := titleRE.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(strings.Join(strings.Fields(m[1]), " "))
}

func (d BrowseDiagnostics) LogFields() map[string]any {
	return map[string]any{
		"html_bytes":               d.HTMLBytes,
		"title":                    d.Title,
		"page_kind":                d.PageKind,
		"subcategory_anchors":      d.SubcategoryAnchorCount,
		"product_anchors":          d.ProductAnchorCount,
		"catalog_href_count":       d.CatalogHrefCount,
		"product_href_count":       d.ProductHrefCount,
		"has_subcategory_block":    d.HasSubcategoryBlock,
		"has_products_page":        d.HasProductsPage,
		"has_header_plug":          d.HasHeaderPlug,
	}
}
