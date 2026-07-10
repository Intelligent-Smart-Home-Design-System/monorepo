package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/infra/postgres"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/pipeline"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/sources"
)

func NewScrapeCmd(log zerolog.Logger, m *metrics.Collector) *cobra.Command {
	var cfgFile string
	var sourcesFlag []string
	var pageTypes []string
	var discoveryOnly bool
	var cleanupDiscovery bool

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Scrape pages from tracked tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()
			return scrape(ctx, log, m, cfgFile, sourcesFlag, pageTypes, discoveryOnly, cleanupDiscovery)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sourcesFlag, "sources", nil, "comma-separated list of sources to scrape (e.g., wildberries,sprut)")
	cmd.Flags().StringSliceVar(&pageTypes, "page-types", nil, "comma-separated list of page types (listing, discovery, compatibility)")
	cmd.Flags().BoolVar(&discoveryOnly, "discovery", false, "if true, scrape only discovery pages")
	cmd.Flags().BoolVar(&cleanupDiscovery, "cleanup-discovery", false, "if true, delete discovery tasks that are not in config")

	return cmd
}

func scrape(ctx context.Context, logger zerolog.Logger, m *metrics.Collector, cfgFile string, sourcesFlag, pageTypes []string, discoveryOnly, cleanupDiscovery bool) error {
	var cfg config.Config
	if err := readConfig(cfgFile, &cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	logger.Info().Msgf("rate limit from config: %f", cfg.Scraping.RateLimitRps)
	logJobStart(logger, "scrape", sourcesFlag, pageTypes, func(e *zerolog.Event) {
		e.Bool("discovery_only", discoveryOnly)
	})

	db, err := postgres.NewDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer db.Close()

	taskRepo := repository.NewTrackedPageRepo(db)
	snapshotRepo := repository.NewSnapshotRepo(db, logger)

	registry, err := sources.NewRegistry(cfg, logger)
	if err != nil {
		return fmt.Errorf("build source registry: %w", err)
	}
	selected := registry.Selected(sourcesFlag)
	scraperMap := registry.ScraperMap()

	if discoveryOnly {
		return (&pipeline.DiscoveryPipeline{
			Log:              logger,
			Metrics:          m,
			Cfg:              cfg,
			Tasks:            taskRepo,
			Snapshots:        snapshotRepo,
			Sources:          selected,
			ScraperMap:       scraperMap,
			CleanupDiscovery: cleanupDiscovery,
		}).Run(ctx)
	}

	return (&pipeline.ListingPipeline{
		Log:         logger,
		Metrics:     m,
		Cfg:         cfg,
		Tasks:       taskRepo,
		Snapshots:   snapshotRepo,
		Sources:     selected,
		ScraperMap:  scraperMap,
		SourceNames: sourcesFlag,
		PageTypes:   pageTypes,
	}).Run(ctx)
}
