package sprut

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const webOrigin = "https://sprut.ai"

// BrowseLinks holds URLs extracted from a sprut.ai catalog page.
type BrowseLinks struct {
	// DiscoveryURLs are subcategory links found on a hub page (h2.header-title a[href]).
	DiscoveryURLs []string
	// ListingURLs are product page links, built from a /catalogs/items API response.
	ListingURLs []string
	// PaginationURLs is unused for sprut: RunDiscoveryBFS resolves every items-API page
	// upfront via _meta.pageCount, so there is no separate pagination-discovery step.
	PaginationURLs []string
}

func (l *BrowseLinks) IsProductGrid() bool {
	return l != nil && len(l.ListingURLs) > 0
}

// extractBrowseLinks parses a sprut.ai hub page (/catalog/section, /catalog/light, ...)
// for subcategory links. Hub pages are server-rendered HTML and load fine directly.
// Leaf/product-grid pages (e.g. /catalog/light/lightbulb) 500 on direct load (Nuxt SSR
// bug on sprut's server) so listing discovery never fetches them as HTML — see
// RunDiscoveryBFS and extractCategoryLinks, which go through api.sprut.ai instead.
func extractBrowseLinks(html []byte, pageURL string) (*BrowseLinks, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	seen := make(map[string]bool)
	var discovery []string

	doc.Find(".card.catalog-row h2.header-title a[href]").Each(func(_ int, s *goquery.Selection) {
		href := normalizeCatalogHref(s.AttrOr("href", ""))
		if href == "" || seen[href] {
			return
		}
		seen[href] = true
		discovery = append(discovery, href)
	})

	if len(discovery) == 0 {
		return nil, fmt.Errorf("no subcategory links found on %s", pageURL)
	}

	return &BrowseLinks{DiscoveryURLs: discovery}, nil
}

// extractCategoryLinks parses a scraped /catalogs/items API response (JSON, saved under
// the "html" resource name like every other page body) into product listing URLs.
func extractCategoryLinks(body []byte, pageURL string) (*BrowseLinks, error) {
	var resp catalogItemsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode catalog items %s: %w", pageURL, err)
	}

	var listings []string
	for _, item := range resp.Items {
		if item.Slug == "" {
			continue
		}
		listings = append(listings, webOrigin+"/catalog/item/"+item.Slug)
	}
	if len(listings) == 0 {
		return nil, fmt.Errorf("no items in catalog page %s", pageURL)
	}

	return &BrowseLinks{ListingURLs: listings}, nil
}

func normalizeCatalogHref(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil || !strings.Contains(u.Host, "sprut.ai") {
			return ""
		}
		raw = u.Path
	}
	if idx := strings.IndexAny(raw, "?#"); idx != -1 {
		raw = raw[:idx]
	}
	if !strings.HasPrefix(raw, "/catalog/") {
		return ""
	}
	return webOrigin + raw
}

func lastPathSegment(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	path := strings.TrimRight(u.Path, "/")
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}
