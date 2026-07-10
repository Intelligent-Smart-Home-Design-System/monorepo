package sources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	dnsParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/dns"
)

type fixtureScraper struct {
	pages map[string][]byte
}

func (f *fixtureScraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	_ = ctx
	body, ok := f.pages[task.URL]
	if !ok {
		return nil, fmt.Errorf("no fixture for %s", task.URL)
	}
	return &domain.ScrapeResult{
		Resources: []domain.Resource{{Name: "html", URL: task.URL, ResponseBody: body, StatusCode: 200}},
	}, nil
}

func loadDNSParserFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "parsers", "dns", "testdata", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

// ExpandDiscovery delegates to parsers/dns.RunDiscoveryBFS — test that path directly with fixtures.
func TestDNS_ExpandDiscovery_ProductGrid(t *testing.T) {
	const gridURL = "https://www.dns-shop.ru/catalog/17a8a01d16404e77/datchiki-protechki/"

	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceDns: {DiscoveryBootstrap: []string{"seed"}},
			},
		},
		Dns: config.DnsConfig{DiscoverySeeds: []string{gridURL}},
		Scraping: config.ScrapingConfig{RateLimitRps: 0},
	}

	scraper := &fixtureScraper{pages: map[string][]byte{
		gridURL: loadDNSParserFixture(t, "category_leak_sensors.html"),
	}}

	var repo MemTaskRepo
	stats, err := dnsParser.RunDiscoveryBFS(
		context.Background(), zerolog.Nop(), cfg.Scraping, cfg.Dns,
		cfg.Dns.DiscoverySeeds, &repo, scraper,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.CategoriesCreated)
	require.Len(t, repo.Seeds, 1)
	assert.Equal(t, domain.PageTypeCategory, repo.Seeds[0].PageType)
	assert.Equal(t, gridURL, repo.Seeds[0].URL)
}

func TestDNS_ExpandDiscovery_HubInMemoryOnly(t *testing.T) {
	const hubURL = "https://www.dns-shop.ru/catalog/17a8a01d16404e77/umnaya-tehnika/"

	cfg := config.Config{
		Dns:      config.DnsConfig{DiscoverySeeds: []string{hubURL}},
		Scraping: config.ScrapingConfig{RateLimitRps: 0},
	}
	scraper := &fixtureScraper{pages: map[string][]byte{
		hubURL: loadDNSParserFixture(t, "category_umnaa_tehnika.html"),
	}}

	var repo MemTaskRepo
	stats, err := dnsParser.RunDiscoveryBFS(
		context.Background(), zerolog.Nop(), cfg.Scraping, cfg.Dns,
		cfg.Dns.DiscoverySeeds, &repo, scraper,
	)
	require.NoError(t, err)
	assert.Empty(t, repo.Seeds)
	assert.Equal(t, 1, stats.HubsVisited)
	assert.Greater(t, stats.InMemoryEnqueued, 5)
}
