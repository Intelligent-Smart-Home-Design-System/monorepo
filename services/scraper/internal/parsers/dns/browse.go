package dns

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const dnsOrigin = "https://www.dns-shop.ru"

// BrowseLinks holds URLs extracted from a DNS catalog/search page.
type BrowseLinks struct {
	// DiscoveryURLs are catalog hub links (subcategories) — enqueued as page_type=discovery.
	DiscoveryURLs []string
	// ListingURLs are /product/... links — enqueued as page_type=listing.
	ListingURLs []string
	// PaginationURLs are additional product-grid pages — enqueued as page_type=category.
	PaginationURLs []string
}

func (l *BrowseLinks) IsProductGrid() bool {
	return l != nil && len(l.ListingURLs) > 0
}

func extractBrowseLinks(html []byte, pageURL string) (*BrowseLinks, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	discoverySeen := make(map[string]bool)
	listingSeen := make(map[string]bool)
	var discovery, listings []string

	addDiscovery := func(path string) {
		path = normalizePath(path)
		if path == "" || !isCategoryPath(path) {
			return
		}
		full := absoluteURL(path)
		if discoverySeen[full] {
			return
		}
		discoverySeen[full] = true
		discovery = append(discovery, full)
	}

	addListing := func(path string) {
		path = normalizePath(path)
		if path == "" || !isProductPath(path) {
			return
		}
		full := absoluteURL(path)
		if listingSeen[full] {
			return
		}
		listingSeen[full] = true
		listings = append(listings, full)
	}

	doc.Find("a.subcategory__item[href]").Each(func(_ int, s *goquery.Selection) {
		addDiscovery(s.AttrOr("href", ""))
	})

	doc.Find("a.catalog-product__image-link[href]").Each(func(_ int, s *goquery.Selection) {
		addListing(s.AttrOr("href", ""))
	})

	if len(discovery) == 0 && len(listings) == 0 {
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			href := s.AttrOr("href", "")
			switch {
			case isProductPath(href):
				addListing(href)
			case isCategoryPath(href):
				addDiscovery(href)
			}
		})
	}

	if len(discovery) == 0 && len(listings) == 0 {
		return nil, fmt.Errorf("no category or product links found")
	}

	links := &BrowseLinks{
		DiscoveryURLs: discovery,
		ListingURLs:   listings,
	}
	if links.IsProductGrid() {
		baseURL := resolveProductGridBaseURL(string(html), pageURL)
		if baseURL != "" {
			links.PaginationURLs = extractPaginationURLs(doc, baseURL)
		}
	}

	return links, nil
}

// extractCategoryLinks parses a product-grid page (listings + pagination).
func extractCategoryLinks(html []byte, pageURL string) (*BrowseLinks, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	listingSeen := make(map[string]bool)
	var listings []string

	addListing := func(path string) {
		path = normalizePath(path)
		if path == "" || !isProductPath(path) {
			return
		}
		full := absoluteURL(path)
		if listingSeen[full] {
			return
		}
		listingSeen[full] = true
		listings = append(listings, full)
	}

	doc.Find("a.catalog-product__image-link[href]").Each(func(_ int, s *goquery.Selection) {
		addListing(s.AttrOr("href", ""))
	})

	if len(listings) == 0 {
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			if isProductPath(s.AttrOr("href", "")) {
				addListing(s.AttrOr("href", ""))
			}
		})
	}

	if len(listings) == 0 {
		return nil, fmt.Errorf("no product links found")
	}

	baseURL := resolveProductGridBaseURL(string(html), pageURL)
	return &BrowseLinks{
		ListingURLs:    listings,
		PaginationURLs: extractPaginationURLs(doc, baseURL),
	}, nil
}

var filtersResultURLRE = regexp.MustCompile(`"resultUrl"\s*:\s*"([^"]+)"`)

func resolveProductGridBaseURL(html, pageURL string) string {
	if m := filtersResultURLRE.FindStringSubmatch(html); len(m) == 2 {
		if u := absoluteURLFromRaw(m[1]); u != "" {
			return u
		}
	}
	if pageURL != "" {
		if u := absoluteURLFromRaw(pageURL); u != "" {
			return stripPageQuery(u)
		}
	}
	return ""
}

func extractPaginationURLs(doc *goquery.Document, baseURL string) []string {
	if baseURL == "" {
		return nil
	}

	maxPage := 1
	doc.Find("[data-role='pagination-page'][data-page-number]").Each(func(_ int, s *goquery.Selection) {
		n, err := strconv.Atoi(strings.TrimSpace(s.AttrOr("data-page-number", "")))
		if err == nil && n > maxPage {
			maxPage = n
		}
	})
	doc.Find("button[data-role='show-more-btn'][data-page-number]").Each(func(_ int, s *goquery.Selection) {
		n, err := strconv.Atoi(strings.TrimSpace(s.AttrOr("data-page-number", "")))
		if err == nil && n > maxPage {
			maxPage = n
		}
	})

	if maxPage <= 1 {
		return nil
	}

	seen := make(map[string]bool)
	var pages []string
	for p := 2; p <= maxPage; p++ {
		u := buildPaginatedURL(baseURL, p)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		pages = append(pages, u)
	}
	return pages
}

func buildPaginatedURL(baseURL string, page int) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return u.String()
}

func stripPageQuery(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Del("page")
	u.RawQuery = q.Encode()
	return u.String()
}

func absoluteURLFromRaw(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return stripPageQuery(raw)
	}
	return absoluteURL(raw)
}

func normalizePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "javascript:") {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return ""
		}
		if !strings.Contains(u.Host, "dns-shop.ru") {
			return ""
		}
		raw = u.Path
	}
	if !strings.HasPrefix(raw, "/") {
		return ""
	}
	if idx := strings.Index(raw, "?"); idx != -1 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, "#"); idx != -1 {
		raw = raw[:idx]
	}
	return raw
}

func isProductPath(path string) bool {
	path = normalizePath(path)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "product" && len(parts[1]) >= 8
}

func isCategoryPath(path string) bool {
	path = normalizePath(path)
	if !strings.HasPrefix(path, "/catalog/") {
		return false
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 {
		return false
	}
	if parts[1] == "product" || parts[1] == "category" || parts[1] == "recipe" {
		return false
	}
	return len(parts[1]) >= 8 && parts[1] != "product"
}

func absoluteURL(path string) string {
	path = normalizePath(path)
	if path == "" {
		return ""
	}
	return dnsOrigin + path
}
