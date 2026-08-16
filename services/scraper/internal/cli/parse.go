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
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	dnsParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/dns"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/sprut"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/wildberries"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/yandex"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
)

// Подключение нового source в parse: internal/parsers/example/doc.go
// (listing → NewWorker; discovery/category → Worker или ParseCategorySnapshots).

func NewParseCmd(log zerolog.Logger, m *metrics.Collector) *cobra.Command {
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
			return parse(ctx, log, m, cfgFile, sources, pageTypes, discoveryOnly)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sources, "sources", nil, "comma-separated list of sources to parse (e.g., wildberries,yandex)")
	cmd.Flags().StringSliceVar(&pageTypes, "page-types", nil, "comma-separated list of page types (listing, discovery, compatibility)")
	cmd.Flags().BoolVar(&discoveryOnly, "discovery", false, "if true, parse only discovery snapshots")

	return cmd
}

func parse(ctx context.Context, logger zerolog.Logger, m *metrics.Collector, cfgFile string, sources, pageTypes []string, discoveryOnly bool) error {
	var cfg config.Config
	if err := readConfig(cfgFile, &cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	logger.Info().Msgf("rate limit from config: %f", cfg.Scraping.RateLimitRps)
	logJobStart(logger, "parse", sources, pageTypes, func(e *zerolog.Event) {
		e.Bool("discovery_only", discoveryOnly)
	})

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
		sourceBySnapshot := make(map[int]string)
		for _, source := range listingSources {
			batch, err := snapshotRepo.GetUnprocessedSnapshots(ctx, domain.PageTypeListing.String(), source)
			if err != nil {
				return fmt.Errorf("get listing snapshots for %s: %w", source, err)
			}
			before := len(batch)
			m.AddParseSnapshots(ctx, source, domain.PageTypeListing.String(), parseJob, metrics.OutcomeMatched, metrics.FilterStageBefore, int64(before))
			batch = filterSnapshots(batch, parseJob, cfg.Jobs)
			m.AddParseSnapshots(ctx, source, domain.PageTypeListing.String(), parseJob, metrics.OutcomeMatched, metrics.FilterStageAfter, int64(len(batch)))
			logger.Info().
				Str("job", parseJob).
				Str("source", source).
				Int("snapshots_total", before).
				Int("snapshots_matched", len(batch)).
				Msg("parse listing: snapshots after job filters")
			snapshots = append(snapshots, batch...)
			for _, s := range batch {
				sourceBySnapshot[s.ID] = s.SourceName
			}
		}

		listingWorker := parser.NewWorker(logger, domain.PageTypeListing, snapshotRepo, listingParsers)
		listingWorker.UseMetrics(m, parseJob)
		listings := listingWorker.ParseSnapshots(ctx, snapshots)

		logger.Debug().Strs("sources", listingSources).Msgf("parsed %d listings", len(listings))
		saved := 0
		skipped := 0
		for _, listing := range listings {
			source := sourceBySnapshot[listing.PageSnapshotID]
			listingLog := withSource(logger, source)
			if !listing.HasSmartHomeMarkers {
				skipped++
				m.AddParseSnapshots(ctx, source, domain.PageTypeListing.String(), parseJob, metrics.OutcomeSkippedNoMarkers, "", 1)
				listingLog.Info().Int("snapshot_id", listing.PageSnapshotID).Msg("listing snapshot has no smart home markers, skipping")
				continue
			}
			if err := snapshotRepo.SaveListingParseResult(listing); err != nil {
				m.AddParseSnapshots(ctx, source, domain.PageTypeListing.String(), parseJob, metrics.OutcomeSaveError, "", 1)
				listingLog.Error().Err(err).Msg("failed to save listing result")
			} else {
				m.AddParseSnapshots(ctx, source, domain.PageTypeListing.String(), parseJob, metrics.OutcomeSaved, "", 1)
				saved++
			}
		}
		logger.Info().Strs("sources", listingSources).Int("parsed", len(listings)).Int("saved", saved).Int("skipped_markers", skipped).Msg("parse listing summary")
	}

	if shouldRun(domain.PageTypeDiscovery, domain.SourceWildberries) {
		discoveryParsers := []parser.SourceParser[[]string]{
			wildberries.NewDiscoveryParser(),
		}
		discoveryWorker := parser.NewWorker(logger, domain.PageTypeDiscovery, snapshotRepo, discoveryParsers)
		discoveryWorker.UseMetrics(m, parseJob)
		parseJob := config.JobParseDiscovery
		if !discoveryOnly {
			parseJob = config.JobParse
		}
		batch, err := snapshotRepo.GetUnprocessedSnapshots(ctx, domain.PageTypeDiscovery.String(), domain.SourceWildberries)
		if err != nil {
			return fmt.Errorf("get wildberries discovery snapshots: %w", err)
		}
		before := len(batch)
		m.AddParseSnapshots(ctx, domain.SourceWildberries, domain.PageTypeDiscovery.String(), parseJob, metrics.OutcomeMatched, metrics.FilterStageBefore, int64(before))
		batch = filterSnapshots(batch, parseJob, cfg.Jobs)
		m.AddParseSnapshots(ctx, domain.SourceWildberries, domain.PageTypeDiscovery.String(), parseJob, metrics.OutcomeMatched, metrics.FilterStageAfter, int64(len(batch)))
		logger.Info().
			Str("job", parseJob).
			Str("source", domain.SourceWildberries).
			Int("discovery_snapshots_total", before).
			Int("discovery_snapshots_matched", len(batch)).
			Msg("parse discovery: snapshots after job filters")
		discoveryResults := discoveryWorker.ParseSnapshots(ctx, batch)

		logger.Debug().Str("source", domain.SourceWildberries).Msgf("processed %d discovery snapshots", len(discoveryResults))
		for _, urls := range discoveryResults {
			for _, productURL := range urls {
				if err := taskRepo.CreateTask(domain.SourceWildberries, domain.PageTypeListing.String(), productURL); err != nil {
					logger.Error().Err(err).Str("source", domain.SourceWildberries).Str("url", productURL).Msg("failed to create listing task from discovery")
				} else {
					logger.Debug().Str("source", domain.SourceWildberries).Str("url", productURL).Msg("created listing task from discovery")
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
					logger.Error().Err(err).Str("source", domain.SourceWildberries).Str("url", productURL).Msg("failed to create listing task from category")
				} else {
					logger.Debug().Str("source", domain.SourceWildberries).Str("url", productURL).Msg("created listing task from category")
				}
			}
		}
	}

	if shouldRunDNSDiscoveryParse(discoveryOnly, sources) {
		job := config.JobParseDiscovery
		savedListings := parseDNSCategorySnapshots(ctx, logger, m, snapshotRepo, taskRepo, cfg.Jobs, job, true)
		if savedListings >= 0 {
			logger.Info().Str("job", job).Str("source", domain.SourceDns).Int("listings_saved", savedListings).Msg("dns parse --discovery summary")
		}
	}

	if shouldRun(domain.PageTypeCategory, domain.SourceDns) && !discoveryOnly {
		parseDNSCategorySnapshots(ctx, logger, m, snapshotRepo, taskRepo, cfg.Jobs, config.JobParse, false)
	}

	if shouldRunSprutDiscoveryParse(discoveryOnly, sources) {
		job := config.JobParseDiscovery
		savedListings := parseSprutCategorySnapshots(ctx, logger, m, snapshotRepo, taskRepo, cfg.Jobs, job, true)
		if savedListings >= 0 {
			logger.Info().Str("job", job).Str("source", domain.SourceSprut).Int("listings_saved", savedListings).Msg("sprut parse --discovery summary")
		}
	}

	if shouldRun(domain.PageTypeCategory, domain.SourceSprut) && !discoveryOnly {
		parseSprutCategorySnapshots(ctx, logger, m, snapshotRepo, taskRepo, cfg.Jobs, config.JobParse, false)
	}

	if shouldRun(domain.PageTypeCompatibility, domain.SourceYandex) {
		compatibilityParsers := []parser.SourceParser[[]*domain.DirectCompatibilityRecord]{
			yandex.NewCompatibilityParser(cfg.Wildberries.BrandAliases),
		}
		compatibilityWorker := parser.NewWorker(logger, domain.PageTypeCompatibility, snapshotRepo, compatibilityParsers)
		compatRecords := compatibilityWorker.Parse(ctx)

		logger.Debug().Str("source", domain.SourceYandex).Msgf("processed %d compatibility snapshots", len(compatRecords))
		for _, records := range compatRecords {
			for _, rec := range records {
				if err := snapshotRepo.SaveDirectCompatibilityRecord(rec); err != nil {
					logger.Error().Err(err).Str("source", domain.SourceYandex).Msg("failed to save compatibility record")
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

func shouldRunSprutDiscoveryParse(discoveryOnly bool, sources []string) bool {
	if !discoveryOnly {
		return false
	}
	if len(sources) == 0 {
		return true
	}
	for _, s := range sources {
		if s == domain.SourceSprut {
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
	m *metrics.Collector,
	snapshotRepo *repository.SnapshotRepo,
	taskRepo *repository.TrackedPageRepo,
	jobs config.JobsConfig,
	job string,
	listingsOnly bool,
) int {
	allSnapshots, err := snapshotRepo.GetUnprocessedSnapshots(ctx, domain.PageTypeCategory.String(), domain.SourceDns)
	if err != nil {
		logger.Error().Err(err).Str("source", domain.SourceDns).Msg("failed to get DNS category snapshots")
		return -1
	}

	before := len(allSnapshots)
	m.AddParseSnapshots(ctx, domain.SourceDns, domain.PageTypeCategory.String(), job, metrics.OutcomeMatched, metrics.FilterStageBefore, int64(before))
	snapshots := filterSnapshots(allSnapshots, job, jobs)
	m.AddParseSnapshots(ctx, domain.SourceDns, domain.PageTypeCategory.String(), job, metrics.OutcomeMatched, metrics.FilterStageAfter, int64(len(snapshots)))
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
		snapshotLog := withSource(logger, domain.SourceDns).With().Str("page_type", domain.PageTypeCategory.String()).Logger()

		files, err := parser.ExtractArchive(snapshot.WARCBundle)
		if err != nil {
			snapshotLog.Error().Err(err).Int("snapshot_id", snapshot.ID).Str("category_url", snapshot.PageURL).Msg("failed to extract category snapshot")
			m.AddParseSnapshots(ctx, domain.SourceDns, domain.PageTypeCategory.String(), job, metrics.OutcomeParseError, "", 1)
			continue
		}

		links, parseErr := categoryParser.Parse(snapshot.ID, files)
		if err := snapshotRepo.SetProcessed(snapshot.ID); err != nil {
			snapshotLog.Error().Err(err).Int("snapshot_id", snapshot.ID).Msg("failed to mark category snapshot processed")
		}
		if parseErr != nil {
			snapshotLog.Error().Err(parseErr).Int("snapshot_id", snapshot.ID).Str("category_url", snapshot.PageURL).Msg("failed to parse category snapshot")
			m.AddParseSnapshots(ctx, domain.SourceDns, domain.PageTypeCategory.String(), job, metrics.OutcomeParseError, "", 1)
			continue
		}
		m.AddParseSnapshots(ctx, domain.SourceDns, domain.PageTypeCategory.String(), job, metrics.OutcomeParsed, "", 1)

		listingCount := 0
		paginationCount := 0
		if links != nil {
			listingCount = len(links.ListingURLs)
			paginationCount = len(links.PaginationURLs)
			for _, listingURL := range links.ListingURLs {
				if err := taskRepo.CreateTask(domain.SourceDns, domain.PageTypeListing.String(), listingURL); err != nil {
					snapshotLog.Error().Err(err).Str("url", listingURL).Msg("failed to create DNS listing task")
				} else {
					savedListings++
				}
			}
			if !listingsOnly {
				for _, pageURL := range links.PaginationURLs {
					if err := taskRepo.CreateTask(domain.SourceDns, domain.PageTypeCategory.String(), pageURL); err != nil {
						snapshotLog.Error().Err(err).Str("url", pageURL).Msg("failed to create DNS category pagination task")
					} else {
						paginationTasks++
					}
				}
			}
		}

		snapshotLog.Info().
			Int("snapshot_id", snapshot.ID).
			Str("category_url", snapshot.PageURL).
			Int("listings_found", listingCount).
			Int("listings_saved", listingCount).
			Int("pagination_pages", paginationCount).
			Msg("dns category parsed")
	}

	logger.Info().
		Str("job", job).
		Str("source", domain.SourceDns).
		Int("categories_parsed", len(snapshots)).
		Int("listings_saved_total", savedListings).
		Int("pagination_tasks", paginationTasks).
		Msg("dns category parse summary")

	return savedListings
}

// parseSprutCategorySnapshots extracts listing URLs from sprut category (product-grid) snapshots.
// Same listingsOnly split as parseDNSCategorySnapshots — see its comment for the rationale.
func parseSprutCategorySnapshots(
	ctx context.Context,
	logger zerolog.Logger,
	m *metrics.Collector,
	snapshotRepo *repository.SnapshotRepo,
	taskRepo *repository.TrackedPageRepo,
	jobs config.JobsConfig,
	job string,
	listingsOnly bool,
) int {
	allSnapshots, err := snapshotRepo.GetUnprocessedSnapshots(ctx, domain.PageTypeCategory.String(), domain.SourceSprut)
	if err != nil {
		logger.Error().Err(err).Str("source", domain.SourceSprut).Msg("failed to get sprut category snapshots")
		return -1
	}

	before := len(allSnapshots)
	m.AddParseSnapshots(ctx, domain.SourceSprut, domain.PageTypeCategory.String(), job, metrics.OutcomeMatched, metrics.FilterStageBefore, int64(before))
	snapshots := filterSnapshots(allSnapshots, job, jobs)
	m.AddParseSnapshots(ctx, domain.SourceSprut, domain.PageTypeCategory.String(), job, metrics.OutcomeMatched, metrics.FilterStageAfter, int64(len(snapshots)))
	logger.Info().
		Str("job", job).
		Str("source", domain.SourceSprut).
		Int("category_snapshots_total", before).
		Int("category_snapshots_matched", len(snapshots)).
		Msg("parse category: snapshots after job filters")

	if len(snapshots) == 0 {
		return -1
	}

	categoryParser := sprut.NewCategoryParser()
	savedListings := 0
	paginationTasks := 0

	for _, snapshot := range snapshots {
		if ctx.Err() != nil {
			break
		}
		snapshotLog := withSource(logger, domain.SourceSprut).With().Str("page_type", domain.PageTypeCategory.String()).Logger()

		files, err := parser.ExtractArchive(snapshot.WARCBundle)
		if err != nil {
			snapshotLog.Error().Err(err).Int("snapshot_id", snapshot.ID).Str("category_url", snapshot.PageURL).Msg("failed to extract category snapshot")
			m.AddParseSnapshots(ctx, domain.SourceSprut, domain.PageTypeCategory.String(), job, metrics.OutcomeParseError, "", 1)
			continue
		}

		links, parseErr := categoryParser.Parse(snapshot.ID, files)
		if err := snapshotRepo.SetProcessed(snapshot.ID); err != nil {
			snapshotLog.Error().Err(err).Int("snapshot_id", snapshot.ID).Msg("failed to mark category snapshot processed")
		}
		if parseErr != nil {
			snapshotLog.Error().Err(parseErr).Int("snapshot_id", snapshot.ID).Str("category_url", snapshot.PageURL).Msg("failed to parse category snapshot")
			m.AddParseSnapshots(ctx, domain.SourceSprut, domain.PageTypeCategory.String(), job, metrics.OutcomeParseError, "", 1)
			continue
		}
		m.AddParseSnapshots(ctx, domain.SourceSprut, domain.PageTypeCategory.String(), job, metrics.OutcomeParsed, "", 1)

		listingCount := 0
		paginationCount := 0
		if links != nil {
			listingCount = len(links.ListingURLs)
			paginationCount = len(links.PaginationURLs)
			for _, listingURL := range links.ListingURLs {
				if err := taskRepo.CreateTask(domain.SourceSprut, domain.PageTypeListing.String(), listingURL); err != nil {
					snapshotLog.Error().Err(err).Str("url", listingURL).Msg("failed to create sprut listing task")
				} else {
					savedListings++
				}
			}
			if !listingsOnly {
				for _, pageURL := range links.PaginationURLs {
					if err := taskRepo.CreateTask(domain.SourceSprut, domain.PageTypeCategory.String(), pageURL); err != nil {
						snapshotLog.Error().Err(err).Str("url", pageURL).Msg("failed to create sprut category pagination task")
					} else {
						paginationTasks++
					}
				}
			}
		}

		snapshotLog.Info().
			Int("snapshot_id", snapshot.ID).
			Str("category_url", snapshot.PageURL).
			Int("listings_found", listingCount).
			Int("listings_saved", listingCount).
			Int("pagination_pages", paginationCount).
			Msg("sprut category parsed")
	}

	logger.Info().
		Str("job", job).
		Str("source", domain.SourceSprut).
		Int("categories_parsed", len(snapshots)).
		Int("listings_saved_total", savedListings).
		Int("pagination_tasks", paginationTasks).
		Msg("sprut category parse summary")

	return savedListings
}
