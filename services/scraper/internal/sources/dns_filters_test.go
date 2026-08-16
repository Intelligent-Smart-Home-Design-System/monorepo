package sources

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/filters"
)

func TestDNSFilters_DiscoveryPhases(t *testing.T) {
	seedOnly := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceDns: {DiscoveryBootstrap: []string{"seed"}},
			},
		},
		Dns: config.DnsConfig{DiscoverySeeds: []string{"https://www.dns-shop.ru/catalog/hub/"}},
	}
	src := dnsSource(t, seedOnly)
	require.NotEmpty(t, src.BootstrapDiscovery(seedOnly))
	assert.Equal(t, []domain.PageType{domain.PageTypeCategory}, src.DiscoveryScrapeTypes(seedOnly))

	dbOnly := seedOnly
	dbOnly.Jobs.ScrapeDiscovery[domain.SourceDns] = config.SourceJobFilter{DiscoveryBootstrap: []string{"db"}}
	src = dnsSource(t, dbOnly)
	assert.Nil(t, src.BootstrapDiscovery(dbOnly))
	assert.Equal(t, []domain.PageType{domain.PageTypeDiscovery, domain.PageTypeCategory},
		src.DiscoveryScrapeTypes(dbOnly))
}

func TestDNSFilters_ScrapeTasksAfterExpand(t *testing.T) {
	repo := &MemTaskRepo{}
	seeds := []TaskSeed{
		{Source: domain.SourceDns, PageType: domain.PageTypeCategory, URL: "https://www.dns-shop.ru/catalog/zigbee/"},
		{Source: domain.SourceDns, PageType: domain.PageTypeCategory, URL: "https://www.dns-shop.ru/catalog/televizory/"},
		{Source: domain.SourceDns, PageType: domain.PageTypeCategory, URL: "https://www.dns-shop.ru/catalog/zigbee/page-2/"},
	}
	require.NoError(t, PersistSeeds(context.Background(), repo, seeds))

	tasks, err := repo.GetTasks(domain.SourceDns, domain.PageTypeCategory.String(), 0)
	require.NoError(t, err)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := range tasks {
		tasks[i].ID = i + 1
		tasks[i].FirstSeenAt = start.Add(time.Duration(i) * time.Hour)
	}

	jobs := config.JobsConfig{
		ScrapeDiscovery: map[string]config.SourceJobFilter{
			domain.SourceDns: {
				URLContains:  []string{"zigbee"},
				CreatedAfter: start.Add(30 * time.Minute),
				Limit:        1,
			},
		},
	}
	matched := filters.ScrapeTasks(tasks, config.JobScrapeDiscovery, jobs)
	require.Len(t, matched, 1)
	assert.Contains(t, matched[0].URL, "zigbee")
	assert.Contains(t, matched[0].URL, "page-2")
}
