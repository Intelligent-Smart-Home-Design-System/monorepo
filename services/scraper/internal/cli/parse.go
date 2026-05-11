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
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/infra/postgres"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/wildberries"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/yandex"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
)

func NewParseCmd() *cobra.Command {
	var cfgFile string
	var sources []string
	var pageTypes []string
	var discoveryOnly bool

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Run the parsing job",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()
			return parse(ctx, cfgFile, sources, pageTypes, discoveryOnly)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sources, "sources", nil, "comma-separated list of sources to parse (e.g., wildberries,yandex)")
	cmd.Flags().StringSliceVar(&pageTypes, "page-types", nil, "comma-separated list of page types (listing, discovery, compatibility)")
	cmd.Flags().BoolVar(&discoveryOnly, "discovery", false, "if true, parse only discovery snapshots")

	return cmd
}

func parse(ctx context.Context, cfgFile string, sources, pageTypes []string, discoveryOnly bool) error {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	var cfg config.Config
	if err := readConfig(cfgFile, &cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	logger.Info().Msgf("rate limit from config: %f", cfg.Scraping.RateLimitRps)

	db, err := postgres.NewDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer db.Close()

	snapshotRepo := repository.NewSnapshotRepo(db, logger)
	taskRepo := repository.NewTrackedPageRepo(db)

	shouldRun := func(pageType domain.PageType, source string) bool {
		if discoveryOnly && pageType != domain.PageTypeDiscovery {
			return false
		}
		if len(pageTypes) > 0 {
			found := false
			for _, pt := range pageTypes {
				if pt == pageType.String() {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		if len(sources) > 0 {
			found := false
			for _, s := range sources {
				if s == source {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	if shouldRun(domain.PageTypeListing, domain.SourceWildberries) {
		listingParsers := []parser.SourceParser[*domain.ListingParseResult]{
			wildberries.NewListingParser(cfg.Wildberries.BrandAliases, cfg.Wildberries.SmartHomeDeviceMarkers),
		}
		listingWorker := parser.NewWorker(logger, domain.PageTypeListing, snapshotRepo, listingParsers)
		listings := listingWorker.Parse(ctx)

		logger.Debug().Msgf("parsed %d listings", len(listings))
		for _, listing := range listings {
			if err := snapshotRepo.SaveListingParseResult(listing); err != nil {
				logger.Error().Err(err).Msg("failed to save listing result")
			}
		}
	}

	if shouldRun(domain.PageTypeDiscovery, domain.SourceWildberries) {
		discoveryParsers := []parser.SourceParser[[]string]{
			wildberries.NewDiscoveryParser(),
		}
		discoveryWorker := parser.NewWorker(logger, domain.PageTypeDiscovery, snapshotRepo, discoveryParsers)
		discoveryResults := discoveryWorker.Parse(ctx)

		logger.Debug().Msgf("processed %d discovery snapshots", len(discoveryResults))
		for _, urls := range discoveryResults {
			for _, productURL := range urls {
				if err := taskRepo.CreateTask(domain.SourceWildberries, domain.PageTypeListing.String(), productURL); err != nil {
					logger.Error().Err(err).Str("url", productURL).Msg("failed to create listing task from discovery")
				} else {
					logger.Debug().Str("url", productURL).Msg("created listing task from discovery")
				}
			}
		}
	}

	if shouldRun(domain.PageTypeCompatibility, domain.SourceYandex) {
		compatibilityParsers := []parser.SourceParser[[]*domain.DirectCompatibilityRecord]{
			yandex.NewCompatibilityParser(cfg.Wildberries.BrandAliases),
		}
		compatibilityWorker := parser.NewWorker(logger, domain.PageTypeCompatibility, snapshotRepo, compatibilityParsers)
		compatRecords := compatibilityWorker.Parse(ctx)

		logger.Debug().Msgf("processed %d compatibility snapshots", len(compatRecords))
		for _, records := range compatRecords {
			for _, rec := range records {
				if err := snapshotRepo.SaveDirectCompatibilityRecord(rec); err != nil {
					logger.Error().Err(err).Msg("failed to save compatibility record")
				}
			}
		}
	}

	return nil
}
