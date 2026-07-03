package dns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/rs/zerolog"
)

// Live probe against dns-shop.ru.
//
// Run (saves HTML + markdown report under testdata/probe/):
//
//	cd services/scraper
//	go test -v -count=1 -run TestProbeDNSFetch ./internal/scrapers/dns/
//
// Custom URL:
//
//	DNS_PROBE_URL='https://www.dns-shop.ru/search/?q=zigbee&page=1' go test -v -run TestProbeDNSFetch ./internal/scrapers/dns/
//
// Skipped in CI short mode: go test -short ./...
func TestProbeDNSFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode; run without -short for live DNS probe")
	}

	urls := probeURLs(t)
	outDir := filepath.Join("testdata", "probe")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir probe output: %v", err)
	}

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	scraper := NewScraper(zerolog.Nop(), 60*time.Second, "", userAgent, nil)

	var combined strings.Builder
	combined.WriteString("# DNS fetch probe report\n\n")
	combined.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	for i, pageURL := range urls {
		t.Logf("fetching %s", pageURL)

		body, status, contentType, err := fetchRaw(t, scraper, pageURL)
		if err != nil {
			t.Errorf("fetch %s: %v", pageURL, err)
			continue
		}

		slug := probeSlug(pageURL, i)
		htmlPath := filepath.Join(outDir, slug+".html")
		mdPath := filepath.Join(outDir, slug+".md")

		if err := os.WriteFile(htmlPath, body, 0o644); err != nil {
			t.Fatalf("write html: %v", err)
		}

		result := AnalyzeProbeResponse(pageURL, status, contentType, body)
		report := result.ReportMarkdown()
		if err := os.WriteFile(mdPath, []byte(report), 0o644); err != nil {
			t.Fatalf("write report: %v", err)
		}

		combined.WriteString(report)
		combined.WriteString("\n---\n\n")

		t.Logf("status=%d bytes=%d case=%s title=%q", status, len(body), result.Case, result.Title)
		t.Logf("saved: %s", htmlPath)
		t.Logf("report: %s", mdPath)
		for _, rec := range result.Recommendations {
			t.Logf("  → %s", rec)
		}
	}

	summaryPath := filepath.Join(outDir, "SUMMARY.md")
	if err := os.WriteFile(summaryPath, []byte(combined.String()), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	t.Logf("summary: %s", summaryPath)
	t.Logf("Open HTML files in a browser to visually inspect what plain HTTP returned.")
}

func probeURLs(t *testing.T) []string {
	if u := os.Getenv("DNS_PROBE_URL"); u != "" {
		return []string{u}
	}
	return []string{
		"https://www.dns-shop.ru/search/?q=" + url.QueryEscape("умный дом") + "&page=1",
		"https://www.dns-shop.ru/search/?q=zigbee&page=1",
		"https://www.dns-shop.ru/product/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/",
	}
}

func probeSlug(rawURL string, index int) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := strings.Trim(re.ReplaceAllString(rawURL, "_"), "_")
	if len(slug) > 80 {
		slug = slug[:80]
	}
	return fmt.Sprintf("%02d_%s", index, slug)
}

func fetchRaw(t *testing.T, scraper *Scraper, pageURL string) ([]byte, int, string, error) {
	t.Helper()

	// Use scraper path first (same as production).
	result, err := scraper.Scrape(context.Background(), domain.ScrapeTask{URL: pageURL})
	if err == nil && len(result.Resources) > 0 {
		r := result.Resources[0]
		return r.ResponseBody, r.StatusCode, r.ResponseHeaders.Get("Content-Type"), nil
	}

	// On non-200 scraper returns error — still capture body for diagnosis.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, 0, "", err
	}
	req.Header.Set("User-Agent", scraper.userAgent)

	resp, err := scraper.client.Do(req)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, resp.Header.Get("Content-Type"), readErr
	}
	return body, resp.StatusCode, resp.Header.Get("Content-Type"), err
}
