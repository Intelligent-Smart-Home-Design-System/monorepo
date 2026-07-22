package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	var retryFailed bool
	var retrySince time.Duration

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Scrape pages from tracked tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()
			return scrape(ctx, log, m, cfgFile, sourcesFlag, pageTypes, discoveryOnly, cleanupDiscovery, retryFailed, retrySince)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sourcesFlag, "sources", nil, "comma-separated list of sources to scrape (e.g., wildberries,sprut)")
	cmd.Flags().StringSliceVar(&pageTypes, "page-types", nil, "comma-separated list of page types (listing, discovery, compatibility)")
	cmd.Flags().BoolVar(&discoveryOnly, "discovery", false, "if true, scrape only discovery pages")
	cmd.Flags().BoolVar(&cleanupDiscovery, "cleanup-discovery", false, "if true, delete discovery tasks that are not in config")
	cmd.Flags().BoolVar(&retryFailed, "retry-failed", false, "if true, only retry pages deactivated by repeated scrape failures instead of the normal queue (defaults to scraping.retry_failed in config)")
	cmd.Flags().DurationVar(&retrySince, "retry-since", 0, "with retry-failed, only retry pages whose last attempt is within this window (defaults to scraping.retry_since in config, or 7 days)")

	return cmd
}

func scrape(ctx context.Context, logger zerolog.Logger, m *metrics.Collector, cfgFile string, sourcesFlag, pageTypes []string, discoveryOnly, cleanupDiscovery, retryFailedFlag bool, retrySinceFlag time.Duration) error {
	var cfg config.Config
	if err := readConfig(cfgFile, &cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// Flags are overrides; the config file is the default source of truth so
	// a full run (including retry-failed) can be driven by --config alone.
	retryFailed := cfg.Scraping.RetryFailed || retryFailedFlag
	retrySince := cfg.Scraping.RetrySince
	if retrySinceFlag != 0 {
		retrySince = retrySinceFlag
	}
	if retrySince == 0 {
		retrySince = 7 * 24 * time.Hour
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
		RetryFailed: retryFailed,
		RetrySince:  time.Now().Add(-retrySince),
	}).Run(ctx)
}
