package dns

import (
	"fmt"
	"regexp"
	"strings"
)

// FetchCase classifies what plain HTTP returned from dns-shop.ru.
type FetchCase string

const (
	FetchCaseBlocked          FetchCase = "blocked_or_error"    // 403, captcha page, access denied
	FetchCaseEmptyShell       FetchCase = "spa_empty_shell"     // minimal HTML, no product data
	FetchCaseSSRWithLinks     FetchCase = "ssr_with_products"   // product URLs/names in raw HTML
	FetchCaseSSRHybrid        FetchCase = "ssr_hybrid_js_prices" // products in HTML, prices via JS/AjaxState
	FetchCaseEmbeddedJSON     FetchCase = "embedded_json_data"  // JSON/state in script tags only
	FetchCaseUnknown          FetchCase = "unknown"
)

type ProbeResult struct {
	URL            string
	StatusCode     int
	ContentType    string
	BodyBytes      int
	Title          string
	Case           FetchCase
	CaseReason     string
	CatalogLinks   []string
	ProductLinks   []string
	Signals        map[string]bool
	BodyPreview    string
	Recommendations []string
}

var (
	titleRe   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	catalogRe = regexp.MustCompile(`/catalog/[a-z0-9\-]+/[a-z0-9\-]+`)
	productRe = regexp.MustCompile(`/product/[0-9a-f]+/[a-z0-9\-]+/`)
)

func AnalyzeProbeResponse(url string, statusCode int, contentType string, body []byte) ProbeResult {
	text := string(body)
	lower := strings.ToLower(text)

	signals := map[string]bool{
		"has_catalog_links":     catalogRe.MatchString(text),
		"has_product_links":     productRe.MatchString(text),
		"has_nuxt_state":        strings.Contains(lower, "__nuxt__") || strings.Contains(lower, "window.__nuxt"),
		"has_react_root":        strings.Contains(lower, `id="root"`) || strings.Contains(lower, `id="__next"`),
		"has_json_ld":           strings.Contains(lower, "application/ld+json"),
		"has_product_microdata": strings.Contains(lower, "itemtype=\"http://schema.org/product\"") ||
			strings.Contains(lower, "itemtype=\"https://schema.org/product\""),
		"has_captcha_page":      strings.Contains(lower, "g-recaptcha") || strings.Contains(lower, "cf-challenge") ||
			strings.Contains(lower, "ddos-guard") || strings.Contains(lower, "please verify you are a human"),
		"has_access_denied": strings.Contains(lower, "access denied") || strings.Contains(lower, "доступ запрещ"),
		"has_price_rub":     strings.Contains(lower, "₽") || strings.Contains(lower, "руб"),
		"has_search_results": strings.Contains(lower, "search-results") || strings.Contains(lower, "catalog-product"),
		"has_ajax_state":     strings.Contains(lower, "window.ajaxstate") || strings.Contains(lower, "ajax-state"),
		"has_lazy_prices":    strings.Contains(lower, "product-buy") && !strings.Contains(lower, "₽"),
	}

	title := extractTitle(text)
	catalogLinks := uniqueMatches(catalogRe.FindAllString(text, 20))
	productLinks := uniqueMatches(productRe.FindAllString(text, 20))

	result := ProbeResult{
		URL:          url,
		StatusCode:   statusCode,
		ContentType:  contentType,
		BodyBytes:    len(body),
		Title:        title,
		Signals:      signals,
		CatalogLinks: catalogLinks,
		ProductLinks: productLinks,
		BodyPreview:  previewBody(text, 2500),
	}

	result.Case, result.CaseReason = classifyFetch(statusCode, signals, len(body), len(catalogLinks), len(productLinks))
	result.Recommendations = recommend(result.Case, signals)
	return result
}

func extractTitle(html string) string {
	m := titleRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(strings.Join(strings.Fields(m[1]), " "))
}

func uniqueMatches(matches []string) []string {
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

func classifyFetch(status int, signals map[string]bool, bodyLen, catalogLinkCount, productLinkCount int) (FetchCase, string) {
	if status == 403 || status == 429 || status >= 500 {
		return FetchCaseBlocked, fmt.Sprintf("HTTP %d", status)
	}
	if signals["has_captcha_page"] || signals["has_access_denied"] {
		return FetchCaseBlocked, "captcha or access-denied page"
	}
	if bodyLen < 8_000 && productLinkCount == 0 && catalogLinkCount == 0 {
		return FetchCaseEmptyShell, fmt.Sprintf("small body (%d bytes) without product links", bodyLen)
	}
	if productLinkCount > 0 || catalogLinkCount > 0 {
		if signals["has_lazy_prices"] || signals["has_ajax_state"] {
			return FetchCaseSSRHybrid, fmt.Sprintf(
				"%d product link(s) in HTML; prices/availability likely loaded by JS (AjaxState)",
				productLinkCount+catalogLinkCount,
			)
		}
		return FetchCaseSSRWithLinks, fmt.Sprintf("found %d product/catalog path(s) in raw HTML", productLinkCount+catalogLinkCount)
	}
	if signals["has_nuxt_state"] || signals["has_json_ld"] {
		return FetchCaseEmbeddedJSON, "structured data in page source but no product links"
	}
	if signals["has_nuxt_state"] || signals["has_react_root"] {
		return FetchCaseEmptyShell, "SPA shell markers without product links in HTML"
	}
	return FetchCaseUnknown, "could not classify confidently"
}

func recommend(caseName FetchCase, signals map[string]bool) []string {
	switch caseName {
	case FetchCaseSSRWithLinks:
		return []string{
			"Plain HTTP + goquery on saved HTML works for discovery (names, URLs).",
			"Save HTML fixtures and write discovery_parser_test.go against them.",
		}
	case FetchCaseSSRHybrid:
		return []string{
			"Product cards are in SSR HTML — discovery/listing names and URLs can be parsed without browser.",
			"Prices and buy buttons load via AjaxState/JS — either call internal API or use headless browser for price.",
			"Fix discovery regex: DNS product URLs are /product/{id}/{slug}/ not /catalog/...",
		}
	case FetchCaseEmbeddedJSON:
		return []string{
			"Inspect saved HTML for JSON blobs (script tags, window.__STATE__).",
			"Prefer parsing embedded JSON or internal API over regex on DOM.",
		}
	case FetchCaseEmptyShell:
		return []string{
			"Plain HTTP is NOT enough — page needs JavaScript rendering.",
			"Use headless browser (rod, like Wildberries session mining) or find XHR/API endpoints.",
		}
	case FetchCaseBlocked:
		return []string{
			"Need realistic User-Agent, cookies, delays, or proxy.",
			"Try headless browser with stealth; inspect Network tab for API.",
		}
	default:
		rec := []string{"Open saved .html in browser and DevTools Network tab."}
		if signals["has_price_rub"] {
			rec = append(rec, "Prices present in source — listing parser may work on static HTML.")
		}
		return rec
	}
}

func previewBody(text string, maxRunes int) string {
	compact := strings.Join(strings.Fields(text), " ")
	if len(compact) <= maxRunes {
		return compact
	}
	return compact[:maxRunes] + "…"
}

func (r ProbeResult) ReportMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# DNS probe: %s\n\n", r.URL)
	fmt.Fprintf(&b, "| Field | Value |\n|-------|-------|\n")
	fmt.Fprintf(&b, "| Status | %d |\n", r.StatusCode)
	fmt.Fprintf(&b, "| Content-Type | %s |\n", r.ContentType)
	fmt.Fprintf(&b, "| Body size | %d bytes |\n", r.BodyBytes)
	fmt.Fprintf(&b, "| Title | %s |\n", r.Title)
	fmt.Fprintf(&b, "| **Case** | **%s** |\n", r.Case)
	fmt.Fprintf(&b, "| Reason | %s |\n\n", r.CaseReason)

	b.WriteString("## Signals\n\n")
	for k, v := range r.Signals {
		fmt.Fprintf(&b, "- %s: %v\n", k, v)
	}

	if len(r.ProductLinks) > 0 {
		b.WriteString("\n## Sample /product/ paths\n\n")
		limit := len(r.ProductLinks)
		if limit > 10 {
			limit = 10
		}
		for _, link := range r.ProductLinks[:limit] {
			fmt.Fprintf(&b, "- `%s`\n", link)
		}
	}

	if len(r.CatalogLinks) > 0 {
		b.WriteString("\n## Sample /catalog/ paths\n\n")
		for _, link := range r.CatalogLinks {
			fmt.Fprintf(&b, "- `%s`\n", link)
		}
	}

	b.WriteString("\n## Recommendations\n\n")
	for _, rec := range r.Recommendations {
		fmt.Fprintf(&b, "- %s\n", rec)
	}

	b.WriteString("\n## Body preview (whitespace collapsed)\n\n```\n")
	b.WriteString(r.BodyPreview)
	b.WriteString("\n```\n")

	return b.String()
}
