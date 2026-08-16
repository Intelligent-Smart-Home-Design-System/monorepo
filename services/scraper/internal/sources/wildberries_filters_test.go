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

func TestWildberriesFilters_DiscoveryPhases(t *testing.T) {
	seedOnly := config.Config{
		Jobs: config.JobsConfig{
			ScrapeDiscovery: map[string]config.SourceJobFilter{
				domain.SourceWildberries: {DiscoveryBootstrap: []string{"seed"}},
			},
		},
		Wildberries: config.WildberriesConfig{
			Discovery: config.WildberriesDiscoveryConfig{DiscoveryTextQueries: []string{"умный дом"}},
		},
	}
	src := wbSource(t, seedOnly)
	require.NotEmpty(t, src.BootstrapDiscovery(seedOnly))
	assert.Nil(t, src.DiscoveryScrapeTypes(seedOnly))

	dbOnly := seedOnly
	dbOnly.Jobs.ScrapeDiscovery[domain.SourceWildberries] = config.SourceJobFilter{DiscoveryBootstrap: []string{"db"}}
	src = wbSource(t, dbOnly)
	assert.Nil(t, src.BootstrapDiscovery(dbOnly))
	assert.Equal(t, []domain.PageType{domain.PageTypeDiscovery}, src.DiscoveryScrapeTypes(dbOnly))
}

func TestWildberriesFilters_ScrapeTasksAfterBootstrap(t *testing.T) {
	repo := &MemTaskRepo{}
	cfg := config.Config{
		Wildberries: config.WildberriesConfig{
			Discovery: config.WildberriesDiscoveryConfig{
				DiscoveryTextQueries: []string{"умная лампа", "tuya", "старый запрос"},
			},
		},
	}
	seeds := wbSource(t, cfg).BootstrapDiscovery(cfg)
	require.NoError(t, PersistSeeds(context.Background(), repo, seeds))

	tasks, err := repo.GetTasks(domain.SourceWildberries, domain.PageTypeDiscovery.String(), 0)
	require.NoError(t, err)
	require.Len(t, tasks, 3)

	now := time.Now()
	for i := range tasks {
		tasks[i].ID = i + 1
		tasks[i].FirstSeenAt = now
	}

	jobs := config.JobsConfig{
		ScrapeDiscovery: map[string]config.SourceJobFilter{
			domain.SourceWildberries: {URLContains: []string{"tuya"}, Limit: 1},
		},
	}
	matched := filters.ScrapeTasks(tasks, config.JobScrapeDiscovery, jobs)
	require.Len(t, matched, 1)
	assert.Contains(t, matched[0].URL, "tuya")
}
