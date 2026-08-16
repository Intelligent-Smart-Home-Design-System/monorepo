//go:build integration

package sources

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	sprutParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/sprut"
)

// go test -tags integration -v -run TestSprutE2E_DiscoveryBFS ./internal/sources/
func TestSprutE2E_DiscoveryBFS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}

	cfg := config.Config{
		Scraping: config.ScrapingConfig{Timeout: 30 * time.Second, RateLimitRps: 2},
		Sprut:    config.SprutConfig{MaxBFSFetches: 20},
	}

	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true}).With().Timestamp().Logger()
	registry, err := NewRegistry(cfg, log)
	require.NoError(t, err)
	src := registry[domain.SourceSprut]

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var repo MemTaskRepo
	stats, err := sprutParser.RunDiscoveryBFS(
		ctx, log, cfg.Scraping, cfg.Sprut,
		[]string{"https://sprut.ai/catalog/light"},
		&repo, src.Scraper(),
	)
	require.NoError(t, err)
	t.Logf("bfs stats: %+v", stats)
	require.NotEmpty(t, repo.Seeds, "expected category tasks from BFS")

	for _, seed := range repo.Seeds {
		require.Equal(t, domain.SourceSprut, seed.Source)
		require.Equal(t, domain.PageTypeCategory, seed.PageType)
		require.Contains(t, seed.URL, "api.sprut.ai/catalogs/items")
	}

	firstURL := repo.Seeds[0].URL
	result, err := src.Scraper().Scrape(ctx, domain.ScrapeTask{
		Source:   domain.SourceSprut,
		PageType: domain.PageTypeCategory,
		URL:      firstURL,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Resources)

	files := make([]*parser.ArchiveFile, 0, len(result.Resources))
	for _, res := range result.Resources {
		files = append(files, &parser.ArchiveFile{Name: res.Name, Data: res.ResponseBody})
	}
	links, err := sprutParser.NewCategoryParser().Parse(1, files)
	require.NoError(t, err)
	require.NotEmpty(t, links.ListingURLs, "expected product listing URLs from category page")
	require.True(t, strings.Contains(links.ListingURLs[0], "/catalog/item/"))
	t.Logf("first category page: %d listing URLs, e.g. %s", len(links.ListingURLs), links.ListingURLs[0])
}
