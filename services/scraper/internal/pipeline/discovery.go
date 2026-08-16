package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/sources"
)

// DiscoveryPipeline runs scrape --discovery for each Source:
//
//	A BootstrapDiscovery → PersistSeeds
//	B Warmup → ExpandDiscovery → PersistSeeds
//	CleanupDiscovery
//	C Warmup → ScrapePhase per DiscoveryScrapeTypes
type DiscoveryPipeline struct {
	Log              zerolog.Logger
	Metrics          *metrics.Collector
	Cfg              config.Config
	Tasks            *repository.TrackedPageRepo
	Snapshots        *repository.SnapshotRepo
	Sources          []sources.Source
	ScraperMap       map[string]scraper.Scraper
	CleanupDiscovery bool
}

func (p *DiscoveryPipeline) Run(ctx context.Context) error {
	for _, src := range p.Sources {
		if err := p.persist(ctx, src.BootstrapDiscovery(p.Cfg)); err != nil {
			p.Log.Error().Err(err).Str("source", src.Name()).Msg("bootstrap discovery failed")
		}
	}

	for _, src := range p.Sources {
		if err := src.Warmup(ctx); err != nil {
			p.Log.Warn().Err(err).Str("source", src.Name()).Msg("warmup before expand failed")
		}
		expanded, err := src.ExpandDiscovery(ctx, p.Cfg)
		if err != nil {
			p.Log.Error().Err(err).Str("source", src.Name()).Msg("expand discovery failed")
			continue
		}
		if err := p.persist(ctx, expanded); err != nil {
			p.Log.Error().Err(err).Str("source", src.Name()).Msg("persist expand discovery failed")
		}
	}

	for _, src := range p.Sources {
		if err := src.CleanupDiscovery(ctx, p.Tasks, p.Cfg, p.CleanupDiscovery); err != nil {
			p.Log.Error().Err(err).Str("source", src.Name()).Msg("discovery cleanup failed")
		}
	}

	for _, src := range p.Sources {
		if err := src.Warmup(ctx); err != nil {
			p.Log.Warn().Err(err).Str("source", src.Name()).Msg("warmup before scrape failed")
		}
		for _, pageType := range src.DiscoveryScrapeTypes(p.Cfg) {
			if err := ScrapePhase(
				ctx, p.Log, p.Metrics, p.Tasks, p.Snapshots, p.ScraperMap, p.Cfg,
				[]string{src.Name()}, nil, true, pageType.String(),
				false, time.Time{},
			); err != nil {
				return err
			}
		}
	}

	p.Log.Info().Msg("all tasks processed, exiting")
	return nil
}

func (p *DiscoveryPipeline) persist(ctx context.Context, seeds []sources.TaskSeed) error {
	if len(seeds) == 0 {
		return nil
	}
	if err := sources.PersistSeeds(ctx, p.Tasks, seeds); err != nil {
		return fmt.Errorf("persist seeds: %w", err)
	}
	return nil
}
