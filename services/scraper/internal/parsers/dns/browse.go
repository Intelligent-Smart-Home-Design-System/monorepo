package dns

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const dnsOrigin = "https://www.dns-shop.ru"

// BrowseLinks holds URLs found on a DNS catalog/search browse page.
type BrowseLinks struct {
	CategoryURLs []string
	ListingURLs  []string
}

func extractBrowseLinks(html []byte) (*BrowseLinks, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	categorySeen := make(map[string]bool)
	listingSeen := make(map[string]bool)
	var categories, listings []string

	addCategory := func(path string) {
		path = normalizePath(path)
		if path == "" || !isCategoryPath(path) {
			return
		}
		full := absoluteURL(path)
		if categorySeen[full] {
			return
		}
		categorySeen[full] = true
		categories = append(categories, full)
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
		addCategory(s.AttrOr("href", ""))
	})

	doc.Find("a.catalog-product__image-link[href]").Each(func(_ int, s *goquery.Selection) {
		addListing(s.AttrOr("href", ""))
	})

	if len(categories) == 0 && len(listings) == 0 {
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			href := s.AttrOr("href", "")
			switch {
			case isProductPath(href):
				addListing(href)
			case isCategoryPath(href):
				addCategory(href)
			}
		})
	}

	if len(categories) == 0 && len(listings) == 0 {
		return nil, fmt.Errorf("no category or product links found")
	}

	return &BrowseLinks{
		CategoryURLs: categories,
		ListingURLs:  listings,
	}, nil
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
