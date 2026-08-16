package sources

import (
	"context"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
)

// TaskRepo is the subset of tracked-page persistence used by sources and pipeline.
type TaskRepo interface {
	CreateTask(source, pageType, url string) error
	GetTasks(source, pageType string, limit int) ([]domain.ScrapeTask, error)
	DeleteTaskByID(id int) error
}

// TaskSeed is one tracked_pages row produced by a discovery/listing bootstrap step.
type TaskSeed struct {
	Source   string
	PageType domain.PageType
	URL      string
}

// Source is the only contract pipeline/cli use per marketplace.
//
// Discovery always runs the same three steps; sources that don't need a step return nil:
//
//	A BootstrapDiscovery — tasks from config (WB: queries, DNS: catalog seeds + search)
//	B ExpandDiscovery    — in-memory catalog walk (DNS: parsers/dns BFS; WB: no-op)
//	C DiscoveryScrapeTypes — page types to fetch from DB (then generic ScrapePhase)
//
// Warmup delegates to scrapers/*. Warmup before B and C.
type Source interface {
	Name() string
	Scraper() scraper.Scraper
	Warmup(ctx context.Context) error

	BootstrapDiscovery(cfg config.Config) []TaskSeed
	ExpandDiscovery(ctx context.Context, cfg config.Config) ([]TaskSeed, error)
	DiscoveryScrapeTypes(cfg config.Config) []domain.PageType
	CleanupDiscovery(ctx context.Context, repo TaskRepo, cfg config.Config, enabled bool) error

	BootstrapListing(cfg config.Config) []TaskSeed
}

// Base provides no-op defaults for optional source hooks.
type Base struct {
	name    string
	scraper scraper.Scraper
}

func (b Base) Name() string            { return b.name }
func (b Base) Scraper() scraper.Scraper { return b.scraper }
func (b Base) Warmup(context.Context) error {
	return nil
}
func (b Base) BootstrapDiscovery(config.Config) []TaskSeed { return nil }
func (b Base) ExpandDiscovery(context.Context, config.Config) ([]TaskSeed, error) {
	return nil, nil
}
func (b Base) DiscoveryScrapeTypes(config.Config) []domain.PageType { return nil }
func (b Base) CleanupDiscovery(context.Context, TaskRepo, config.Config, bool) error {
	return nil
}
func (b Base) BootstrapListing(config.Config) []TaskSeed { return nil }

func discoveryBootstrapModes(cfg config.Config, source string) (seed, db bool) {
	filter := filters.JobSourceFilter(cfg.Jobs, config.JobScrapeDiscovery, source)
	return filter.BootstrapMode()
}

// PersistSeeds writes bootstrap results; shared by pipeline for all sources.
func PersistSeeds(ctx context.Context, repo TaskRepo, seeds []TaskSeed) error {
	_ = ctx
	for _, seed := range seeds {
		if err := repo.CreateTask(seed.Source, seed.PageType.String(), seed.URL); err != nil {
			return err
		}
	}
	return nil
}

// MemTaskRepo records CreateTask calls in memory (unit tests).
type MemTaskRepo struct {
	Seeds []TaskSeed
}

func (m *MemTaskRepo) CreateTask(source, pageType, url string) error {
	m.Seeds = append(m.Seeds, TaskSeed{
		Source:   source,
		PageType: domain.PageTypeFromString(pageType),
		URL:      url,
	})
	return nil
}

func (m *MemTaskRepo) GetTasks(source, pageType string, limit int) ([]domain.ScrapeTask, error) {
	_ = limit
	var out []domain.ScrapeTask
	for i, seed := range m.Seeds {
		if source != "" && seed.Source != source {
			continue
		}
		if pageType != "" && seed.PageType.String() != pageType {
			continue
		}
		out = append(out, domain.ScrapeTask{
			ID:       i + 1,
			Source:   seed.Source,
			PageType: seed.PageType,
			URL:      seed.URL,
		})
	}
	return out, nil
}

func (m *MemTaskRepo) DeleteTaskByID(id int) error {
	idx := id - 1
	if idx < 0 || idx >= len(m.Seeds) {
		return nil
	}
	m.Seeds = append(m.Seeds[:idx], m.Seeds[idx+1:]...)
	return nil
}
