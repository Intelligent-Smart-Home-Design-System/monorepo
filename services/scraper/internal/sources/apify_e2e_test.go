//go:build integration

package sources

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

// E2E: real Apify actor run — start → poll → fetch dataset. Needs a real
// token, so it's skipped (not failed) when one isn't set.
//
//	APIFY_API_KEY=your_token APIFY_ACTOR_ID=someuser/some-actor \
//	  go test -tags integration -v -run TestApifyE2E ./internal/sources/
//
// Optional overrides:
//
//	APIFY_E2E_QUERY='умный дом'   # search query, default below
//	APIFY_REGION=213              # default 213 (Moscow)
//	APIFY_MAX_ITEMS=5             # default 5, keep small — real actor run, costs quota
func TestApifyE2E_DiscoveryScrape(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}

	apiKey := os.Getenv("APIFY_API_KEY")
	actorID := os.Getenv("APIFY_ACTOR_ID")
	if apiKey == "" || actorID == "" {
		t.Skip("set APIFY_API_KEY and APIFY_ACTOR_ID to run this test")
	}

	query := os.Getenv("APIFY_E2E_QUERY")
	if query == "" {
		query = "умный дом"
	}

	cfg := config.Config{
		Scraping: config.ScrapingConfig{
			Timeout: 90 * time.Second,
		},
		Apify: config.ApifyConfig{
			APIKey:   apiKey,
			ActorID:  actorID,
			Region:   213,
			MaxItems: 5,
		},
	}

	registry, err := NewRegistry(cfg, zerolog.Nop())
	require.NoError(t, err)
	src := registry[domain.SourceApifyYandexMarket]
	require.NotNil(t, src, "apify source not registered")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := src.Scraper().Scrape(ctx, domain.ScrapeTask{
		Source:   domain.SourceApifyYandexMarket,
		PageType: domain.PageTypeDiscovery,
		URL:      query,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Resources)

	body := result.Resources[0].ResponseBody
	t.Logf("e2e apify: query=%q bytes=%d status=%d", query, len(body), result.Resources[0].StatusCode)
	require.NotEmpty(t, body)
}
