package sources

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

func dnsSource(t *testing.T, cfg config.Config) DNS {
	t.Helper()
	reg, err := NewRegistry(cfg, zerolog.Nop())
	require.NoError(t, err)
	src, ok := reg[domain.SourceDns].(DNS)
	require.True(t, ok)
	return src
}

func TestDNS_BootstrapDiscovery_CatalogAndSearch(t *testing.T) {
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceDns: {DiscoveryBootstrap: []string{"seed"}},
			},
		},
		Dns: config.DnsConfig{
			DiscoverySeeds: []string{"https://www.dns-shop.ru/catalog/17a8a01d16404e77/umnaya-tehnika/"},
			SearchQueries: []string{"zigbee"},
			MaxPages:      2,
		},
	}

	seeds := dnsSource(t, cfg).BootstrapDiscovery(cfg)
	require.Len(t, seeds, 3)
	assert.Equal(t, "https://www.dns-shop.ru/catalog/17a8a01d16404e77/umnaya-tehnika/", seeds[0].URL)
	assert.Equal(t, "https://www.dns-shop.ru/search/?q=zigbee&page=1", seeds[1].URL)
	assert.Equal(t, "https://www.dns-shop.ru/search/?q=zigbee&page=2", seeds[2].URL)
}

func TestDNS_DiscoveryScrapeTypes_SeedOnly(t *testing.T) {
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceDns: {DiscoveryBootstrap: []string{"seed"}},
			},
		},
	}
	assert.Equal(t, []domain.PageType{domain.PageTypeCategory}, dnsSource(t, cfg).DiscoveryScrapeTypes(cfg))
}

func TestDNS_DiscoveryScrapeTypes_SeedAndDb(t *testing.T) {
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceDns: {DiscoveryBootstrap: []string{"seed", "db"}},
			},
		},
	}
	assert.Equal(t, []domain.PageType{domain.PageTypeDiscovery, domain.PageTypeCategory},
		dnsSource(t, cfg).DiscoveryScrapeTypes(cfg))
}

func TestDNS_BootstrapDiscovery_SeedDisabled(t *testing.T) {
	cfg := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceDns: {DiscoveryBootstrap: []string{"db"}},
			},
		},
		Dns: config.DnsConfig{
			DiscoverySeeds: []string{"https://www.dns-shop.ru/catalog/hub/"},
		},
	}
	assert.Nil(t, dnsSource(t, cfg).BootstrapDiscovery(cfg))
}
