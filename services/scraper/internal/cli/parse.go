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
	dnsParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/dns"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/sprut"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/wildberries"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/yandex"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
)

// Подключение нового source в parse: internal/parsers/example/doc.go
// (listing → NewWorker; discovery/category → Worker или ParseCategorySnapshots).

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

	parseJob := config.JobParse
	if discoveryOnly {
		parseJob = config.JobParseDiscovery
	}

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

	if shouldRun(domain.PageTypeListing, domain.SourceWildberries) || shouldRun(domain.PageTypeListing, domain.SourceSprut) || shouldRun(domain.PageTypeListing, domain.SourceDns) {
		var listingParsers []parser.SourceParser[*domain.ListingParseResult]
		var listingSources []string
		if shouldRun(domain.PageTypeListing, domain.SourceWildberries) {
			listingParsers = append(listingParsers,
				wildberries.NewListingParser(cfg.Wildberries.BrandAliases, cfg.Wildberries.SmartHomeDeviceMarkers),
			)
			listingSources = append(listingSources, domain.SourceWildberries)
		}
		if shouldRun(domain.PageTypeListing, domain.SourceSprut) {
			listingParsers = append(listingParsers,
				sprut.NewListingParser(cfg.Wildberries.BrandAliases),
			)
			listingSources = append(listingSources, domain.SourceSprut)
		}
		if shouldRun(domain.PageTypeListing, domain.SourceDns) {
			listingParsers = append(listingParsers,
				dnsParser.NewListingParser(cfg.Dns.BrandAliases, cfg.Dns.SmartHomeDeviceMarkers),
			)
			listingSources = append(listingSources, domain.SourceDns)
		}

		var snapshots []*domain.PageSnapshot
		for _, source := range listingSources {
			batch, err := snapshotRepo.GetUnprocessedSnapshots(ctx, domain.PageTypeListing.String(), source)
			if err != nil {
				return fmt.Errorf("get listing snapshots for %s: %w", source, err)
			}
			before := len(batch)
			batch = filterSnapshots(batch, parseJob, cfg.Jobs)
			logger.Info().
				Str("job", parseJob).
				Str("source", source).
				Int("snapshots_total", before).
				Int("snapshots_matched", len(batch)).
				Msg("parse listing: snapshots after job filters")
			snapshots = append(snapshots, batch...)
		}

		listingWorker := parser.NewWorker(logger, domain.PageTypeListing, snapshotRepo, listingParsers)
		listings := listingWorker.ParseSnapshots(ctx, snapshots)

		logger.Debug().Msgf("parsed %d listings", len(listings))
		saved := 0
		skipped := 0
		for _, listing := range listings {
			if !listing.HasSmartHomeMarkers {
				skipped++
				logger.Info().Int("snapshot_id", listing.PageSnapshotID).Msg("listing snapshot has no smart home markers, skipping")
				continue
			}
			if err := snapshotRepo.SaveListingParseResult(listing); err != nil {
				logger.Error().Err(err).Msg("failed to save listing result")
			} else {
				saved++
			}
		}
		logger.Info().Int("parsed", len(listings)).Int("saved", saved).Int("skipped_markers", skipped).Msg("parse listing summary")
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
					logger.Debug().Str("url", productURL).Msg("created listing task from category")
				}
			}
		}
	}

	if shouldRun(domain.PageTypeCategory, domain.SourceWildberries) {
		categoryParsers := []parser.SourceParser[[]string]{
			wildberries.NewCategoryParser(),
		}
		categoryWorker := parser.NewWorker(logger, domain.PageTypeCategory, snapshotRepo, categoryParsers)
		categoryResults := categoryWorker.Parse(ctx)
		for _, urls := range categoryResults {
			for _, productURL := range urls {
				if err := taskRepo.CreateTask(domain.SourceWildberries, domain.PageTypeListing.String(), productURL); err != nil {
					logger.Error().Err(err).Str("url", productURL).Msg("failed to create listing task from category")
				} else {
					logger.Debug().Str("url", productURL).Msg("created listing task from category")
				}
			}
		}
	}

	if shouldRunDNSDiscoveryParse(discoveryOnly, sources) {
		job := config.JobParseDiscovery
		savedListings := parseDNSCategorySnapshots(ctx, logger, snapshotRepo, taskRepo, cfg.Jobs, job, true)
		if savedListings >= 0 {
			logger.Info().Str("job", job).Int("listings_saved", savedListings).Msg("dns parse --discovery summary")
		}
	}

	if shouldRun(domain.PageTypeCategory, domain.SourceDns) && !discoveryOnly {
		parseDNSCategorySnapshots(ctx, logger, snapshotRepo, taskRepo, cfg.Jobs, config.JobParse, false)
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

func shouldRunDNSDiscoveryParse(discoveryOnly bool, sources []string) bool {
	if !discoveryOnly {
		return false
	}
	if len(sources) == 0 {
		return true
	}
	for _, s := range sources {
		if s == domain.SourceDns {
			return true
		}
	}
	return false
}

// parseDNSCategorySnapshots extracts listing URLs from DNS category snapshots.
//
// listingsOnly — не про «неоднозначные данные», а про разные побочные эффекты одного парсера:
//   - parse --discovery (listingsOnly=true): только CreateTask(listing); пагинацию category не трогаем
//     (следующие страницы каталога уже созданы BFS или отдельным job).
//   - parse --page-types category (listingsOnly=false): ещё CreateTask(category) для PaginationURLs.
//
// Вход всегда один: category snapshot. Выход парсера (*BrowseLinks) тоже один.
// Флаг выбирает, какие поля BrowseLinks превращать в новые tracked_pages.
func parseDNSCategorySnapshots(
	ctx context.Context,
	logger zerolog.Logger,
	snapshotRepo *repository.SnapshotRepo,
	taskRepo *repository.TrackedPageRepo,
	jobs config.JobsConfig,
	job string,
	listingsOnly bool,
) int {
	allSnapshots, err := snapshotRepo.GetUnprocessedSnapshots(ctx, domain.PageTypeCategory.String(), domain.SourceDns)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get DNS category snapshots")
		return -1
	}

	before := len(allSnapshots)
	snapshots := filterSnapshots(allSnapshots, job, jobs)
	logger.Info().
		Str("job", job).
		Str("source", domain.SourceDns).
		Int("category_snapshots_total", before).
		Int("category_snapshots_matched", len(snapshots)).
		Msg("parse category: snapshots after job filters")

	if len(snapshots) == 0 {
		return -1
	}

	categoryParser := dnsParser.NewCategoryParser()
	savedListings := 0
	paginationTasks := 0

	for _, snapshot := range snapshots {
		if ctx.Err() != nil {
			break
		}

		files, err := parser.ExtractArchive(snapshot.WARCBundle)
		if err != nil {
			logger.Error().Err(err).Int("snapshot_id", snapshot.ID).Str("category_url", snapshot.PageURL).Msg("failed to extract category snapshot")
			continue
		}

		links, parseErr := categoryParser.Parse(snapshot.ID, files)
		if err := snapshotRepo.SetProcessed(snapshot.ID); err != nil {
			logger.Error().Err(err).Int("snapshot_id", snapshot.ID).Msg("failed to mark category snapshot processed")
		}
		if parseErr != nil {
			logger.Error().Err(parseErr).Int("snapshot_id", snapshot.ID).Str("category_url", snapshot.PageURL).Msg("failed to parse category snapshot")
			continue
		}

		listingCount := 0
		paginationCount := 0
		if links != nil {
			listingCount = len(links.ListingURLs)
			paginationCount = len(links.PaginationURLs)
			for _, listingURL := range links.ListingURLs {
				if err := taskRepo.CreateTask(domain.SourceDns, domain.PageTypeListing.String(), listingURL); err != nil {
					logger.Error().Err(err).Str("url", listingURL).Msg("failed to create DNS listing task")
				} else {
					savedListings++
				}
			}
			if !listingsOnly {
				for _, pageURL := range links.PaginationURLs {
					if err := taskRepo.CreateTask(domain.SourceDns, domain.PageTypeCategory.String(), pageURL); err != nil {
						logger.Error().Err(err).Str("url", pageURL).Msg("failed to create DNS category pagination task")
					} else {
						paginationTasks++
					}
				}
			}
		}

		logger.Info().
			Int("snapshot_id", snapshot.ID).
			Str("category_url", snapshot.PageURL).
			Int("listings_found", listingCount).
			Int("listings_saved", listingCount).
			Int("pagination_pages", paginationCount).
			Msg("dns category parsed")
	}

	logger.Info().
		Str("job", job).
		Int("categories_parsed", len(snapshots)).
		Int("listings_saved_total", savedListings).
		Int("pagination_tasks", paginationTasks).
		Msg("dns category parse summary")

	return savedListings
}
