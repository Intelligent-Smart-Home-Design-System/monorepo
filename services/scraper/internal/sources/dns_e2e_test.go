package sources

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	dnsParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/dns"
)

// Stable DNS product used in parser fixtures and probe tests (MOES Zigbee leak sensor).
const defaultDNSListingURL = "https://www.dns-shop.ru/product/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/"

// E2E: warmup → scrape listing (html + product-buy + characteristics) → parse → assert price/name.
//
//	go test -v -count=1 -run TestDNSE2E_ListingScrapeParse ./internal/sources/
//
// Override product URL:
//
//	DNS_E2E_LISTING_URL='https://www.dns-shop.ru/product/.../' go test -v -run TestDNSE2E ./internal/sources/
func TestDNSE2E_ListingScrapeParse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}

	listingURL := os.Getenv("DNS_E2E_LISTING_URL")
	if listingURL == "" {
		listingURL = defaultDNSListingURL
	}

	browserUserMode := true
	cfg := config.Config{
		Scraping: config.ScrapingConfig{
			Timeout:      90 * time.Second,
			RateLimitRps: 1,
			UserAgent:    "Mozilla/5.0 (Linux; Android 14; Mobile) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		},
		Dns: config.DnsConfig{
			BrowserUserMode: &browserUserMode,
			UserAgent:       "Mozilla/5.0 (Linux; Android 14; Mobile) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			BrandAliases:    map[string]string{"moes": "moes"},
			SmartHomeDeviceMarkers: []string{
				"Zigbee", "умный дом", "умный", "датчик",
			},
		},
	}
	applyLiveNetworkConfig(&cfg)

	registry, err := NewRegistry(cfg, zerolog.Nop())
	require.NoError(t, err)
	src := registry[domain.SourceDns]

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	require.NoError(t, src.Warmup(ctx))

	result, err := src.Scraper().Scrape(ctx, domain.ScrapeTask{
		Source:   domain.SourceDns,
		PageType: domain.PageTypeListing,
		URL:      listingURL,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Resources)

	resourceNames := make([]string, 0, len(result.Resources))
	files := make([]*parser.ArchiveFile, 0, len(result.Resources))
	for _, res := range result.Resources {
		resourceNames = append(resourceNames, res.Name)
		files = append(files, &parser.ArchiveFile{Name: res.Name, Data: res.ResponseBody})
	}
	t.Logf("e2e scrape: %d resource(s): %v", len(result.Resources), resourceNames)
	assert.Contains(t, resourceNames, "html")

	parsed, err := dnsParser.NewListingParser(cfg.Dns.BrandAliases, cfg.Dns.SmartHomeDeviceMarkers).
		Parse(1, files)
	require.NoError(t, err)

	require.True(t, parsed.HasSmartHomeMarkers, "expected smart-home markers in listing text")
	require.NotEmpty(t, parsed.Name)
	assert.Contains(t, strings.ToLower(parsed.Name), "moes")
	require.NotNil(t, parsed.Price, "price must come from product-buy.json scrape")
	assert.Greater(t, *parsed.Price, 0)
	require.NotNil(t, parsed.Currency)
	assert.Equal(t, "RUB", *parsed.Currency)
	assert.NotEmpty(t, parsed.Brand)
	assert.True(t, parsed.InStock)

	t.Logf("e2e parse: name=%q brand=%q price=%d %s reviews=%d rating=%.2f",
		parsed.Name, parsed.Brand, *parsed.Price, *parsed.Currency,
		parsed.ReviewCount, parsed.Rating)
}
