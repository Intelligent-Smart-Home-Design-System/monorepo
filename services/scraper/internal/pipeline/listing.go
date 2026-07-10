package pipeline

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/sources"
)

// ListingPipeline: BootstrapListing → Warmup → ScrapePhase.
type ListingPipeline struct {
	Log        zerolog.Logger
	Metrics    *metrics.Collector
	Cfg        config.Config
	Tasks      *repository.TrackedPageRepo
	Snapshots  *repository.SnapshotRepo
	Sources    []sources.Source
	ScraperMap map[string]scraper.Scraper
	SourceNames []string
	PageTypes  []string
}

func (p *ListingPipeline) Run(ctx context.Context) error {
	for _, src := range p.Sources {
		if err := sources.PersistSeeds(ctx, p.Tasks, src.BootstrapListing(p.Cfg)); err != nil {
			p.Log.Error().Err(err).Str("source", src.Name()).Msg("bootstrap listing failed")
		}
	}

	for _, src := range p.Sources {
		if err := src.Warmup(ctx); err != nil {
			p.Log.Warn().Err(err).Str("source", src.Name()).Msg("warmup failed")
		}
	}

	pageTypeFilter := discoveryPageType(p.PageTypes, false)
	if err := ScrapePhase(
		ctx, p.Log, p.Metrics, p.Tasks, p.Snapshots, p.ScraperMap, p.Cfg,
		p.SourceNames, p.PageTypes, false, pageTypeFilter,
	); err != nil {
		return err
	}

	p.Log.Info().Msg("all tasks processed, exiting")
	return nil
}
