package sources

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	wbScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
)

const testWBDiscoveryTemplate = "https://www.wildberries.ru/__internal/u-search/exactmatch/ru/common/v18/search?appType=1&page={page}&query={query}&resultset=catalog"

func wbSource(t *testing.T, cfg config.Config) Wildberries {
	t.Helper()
	reg, err := NewRegistry(cfg, zerolog.Nop())
	require.NoError(t, err)
	src, ok := reg[domain.SourceWildberries].(Wildberries)
	require.True(t, ok)
	return src
}

func TestWildberries_BootstrapDiscovery_ExplicitTemplateAndQueries(t *testing.T) {
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceWildberries: {DiscoveryBootstrap: []string{"seed"}},
			},
		},
		Wildberries: config.WildberriesConfig{
			Discovery: config.WildberriesDiscoveryConfig{
				DiscoveryTextQueries: []string{"умная лампа", "zigbee датчик"},
				URLTemplate:          testWBDiscoveryTemplate,
				MaxPages:             3,
			},
		},
	}

	seeds := wbSource(t, cfg).BootstrapDiscovery(cfg)
	require.Len(t, seeds, 2)
	assert.Equal(t, []TaskSeed{
		{Source: domain.SourceWildberries, PageType: domain.PageTypeDiscovery, URL: "wildberries://discovery/умная лампа"},
		{Source: domain.SourceWildberries, PageType: domain.PageTypeDiscovery, URL: "wildberries://discovery/zigbee датчик"},
	}, seeds)

	apiURL := wbScraper.BuildDiscoverySearchURL(testWBDiscoveryTemplate, "умная лампа", 1)
	assert.Equal(
		t,
		"https://www.wildberries.ru/__internal/u-search/exactmatch/ru/common/v18/search?appType=1&page=1&query=%D1%83%D0%BC%D0%BD%D0%B0%D1%8F+%D0%BB%D0%B0%D0%BC%D0%BF%D0%B0&resultset=catalog",
		apiURL,
	)
}

func TestWildberries_BootstrapDiscovery_SeedDisabled(t *testing.T) {
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceWildberries: {DiscoveryBootstrap: []string{"db"}},
			},
		},
		Wildberries: config.WildberriesConfig{
			Discovery: config.WildberriesDiscoveryConfig{DiscoveryTextQueries: []string{"умный дом"}},
		},
	}
	assert.Nil(t, wbSource(t, cfg).BootstrapDiscovery(cfg))
}

func TestWildberries_DiscoveryScrapeTypes_DbOnly(t *testing.T) {
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceWildberries: {DiscoveryBootstrap: []string{"db"}},
			},
		},
	}
	assert.Equal(t, []domain.PageType{domain.PageTypeDiscovery}, wbSource(t, cfg).DiscoveryScrapeTypes(cfg))
}

func TestWildberries_BootstrapListing(t *testing.T) {
	cfg := config.Config{
		Wildberries: config.WildberriesConfig{
			Category: config.WildberriesCategoryConfig{
				CategoryURL: "https://www.wildberries.ru/catalog/elektronika/umnyy-dom",
			},
		},
	}
	seeds := wbSource(t, cfg).BootstrapListing(cfg)
	require.Len(t, seeds, 1)
	assert.Equal(t, domain.PageTypeCategory, seeds[0].PageType)
}

func TestWildberries_ExpandDiscovery_NoOp(t *testing.T) {
	expanded, err := wbSource(t, config.Config{}).ExpandDiscovery(context.Background(), config.Config{})
	require.NoError(t, err)
	assert.Nil(t, expanded)
}
