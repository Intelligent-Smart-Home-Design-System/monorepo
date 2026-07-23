//go:build smoke

package sources

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	dnsParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/dns"
)

// Live smoke tests share one DNS scraper per run to avoid Chrome profile lock between subtests.
func TestDNSSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}

	browserUserMode := true
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceDns: {DiscoveryBootstrap: []string{"seed"}},
			},
		},
		Scraping: config.ScrapingConfig{
			Timeout:      90 * time.Second,
			RateLimitRps: 1,
			UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		Dns: config.DnsConfig{
			BrowserUserMode: &browserUserMode,
			MaxBFSFetches:   SmokeMaxDNSFetches,
		},
	}
	applyLiveNetworkConfig(&cfg)

	registry, err := NewRegistry(cfg, zerolog.Nop())
	require.NoError(t, err)
	src := registry[domain.SourceDns]
	require.NotNil(t, src)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	require.NoError(t, src.Warmup(ctx))

	t.Run("CategoryOneFetch", func(t *testing.T) {
		const categoryURL = "https://www.dns-shop.ru/catalog/17a8a01d16404e77/datchiki-protechki/"
		result, err := src.Scraper().Scrape(ctx, domain.ScrapeTask{
			Source: domain.SourceDns, PageType: domain.PageTypeCategory, URL: categoryURL,
		})
		require.NoError(t, err)
		require.Len(t, result.Resources, 1)
		assert.Greater(t, len(result.Resources[0].ResponseBody), 15_000)
		t.Log("dns category smoke: 1 HTML fetch (after warmup)")
	})

	t.Run("BFSFetchCap", func(t *testing.T) {
		const hubURL = "https://www.dns-shop.ru/catalog/17a8a01d16404e77/umnaya-tehnika/"
		cfg.Dns.DiscoverySeeds = []string{hubURL}

		var repo MemTaskRepo
		stats, err := dnsParser.RunDiscoveryBFS(
			ctx, zerolog.Nop(), cfg.Scraping, cfg.Dns, cfg.Dns.DiscoverySeeds, &repo, src.Scraper(),
		)
		require.NoError(t, err)
		assert.Greater(t, stats.Fetches, 0)
		assert.LessOrEqual(t, stats.Fetches, SmokeMaxDNSFetches)
		t.Logf("dns bfs smoke: %d fetch(es) (cap %d), categories=%d",
			stats.Fetches, SmokeMaxDNSFetches, stats.CategoriesCreated)
	})
}

// DNS live smoke — see config.mobile-proxy.toml for proxy/session paths.
