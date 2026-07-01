package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/infra/postgres"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	dnsParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/dns"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
	dnsScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/dns"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/printer"
	sprutScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/sprut"
	wbScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
	yandexScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/yandex"
)

func NewScrapeCmd(log zerolog.Logger, m *metrics.Collector) *cobra.Command {
	var cfgFile string
	var sources []string
	var pageTypes []string
	var discoveryOnly bool
	var cleanupDiscovery bool

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Scrape pages from tracked tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()
			return scrape(ctx, log, m, cfgFile, sources, pageTypes, discoveryOnly, cleanupDiscovery)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sources, "sources", nil, "comma-separated list of sources to scrape (e.g., wildberries,sprut)")
	cmd.Flags().StringSliceVar(&pageTypes, "page-types", nil, "comma-separated list of page types (listing, discovery, compatibility)")
	cmd.Flags().BoolVar(&discoveryOnly, "discovery", false, "if true, scrape only discovery pages")
	cmd.Flags().BoolVar(&cleanupDiscovery, "cleanup-discovery", false, "if true, delete discovery tasks that are not in config")

	return cmd
}

func scrape(ctx context.Context, logger zerolog.Logger, m *metrics.Collector, cfgFile string, sources, pageTypes []string, discoveryOnly, cleanupDiscovery bool) error {
	var cfg config.Config
	if err := readConfig(cfgFile, &cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	logger.Info().Msgf("rate limit from config: %f", cfg.Scraping.RateLimitRps)
	logJobStart(logger, "scrape", sources, pageTypes, func(e *zerolog.Event) {
		e.Bool("discovery_only", discoveryOnly)
	})

	db, err := postgres.NewDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer db.Close()

	taskRepo := repository.NewTrackedPageRepo(db)
	snapshotRepo := repository.NewSnapshotRepo(db, logger)

	printerScraper := printer.NewPrinterScraper()
	wildberriesScraper := wbScraper.NewScraper(
		logger,
		cfg.Scraping.Timeout,
		cfg.Scraping.Proxy,
		cfg.Scraping.WBCardBasket,
		cfg.Scraping.WBRPS,
		cfg.Scraping.WBSessionPath,
		cfg.Wildberries.Discovery.URLTemplate,
		cfg.Wildberries.Discovery.MaxPages,
	)

	if !discoveryOnly {
		if cfg.Wildberries.Category.CategoryURL != "" {
			if err := taskRepo.CreateTask(domain.SourceWildberries, domain.PageTypeCategory.String(), cfg.Wildberries.Category.CategoryURL); err != nil {
				logger.Error().Err(err).Msg("failed to create category task")
			}
		}
		if cfg.Yandex.SupportedZigbeeDevicesURL != "" {
			if err := taskRepo.CreateTask(domain.SourceYandex, domain.PageTypeCompatibility.String(), cfg.Yandex.SupportedZigbeeDevicesURL); err != nil {
				logger.Error().Err(err).Msg("failed to create Yandex compatibility task")
			}
		}
	}

	dnsScraperInstance := dnsScraper.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.Proxy, cfg.Dns.UserAgent)
	yandexScraperInstance := yandexScraper.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.Proxy, cfg.Scraping.RateLimitRps)
	sprutScraperInstance := sprutScraper.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.UserAgent)

	sourceToScraper := map[string]scraper.Scraper{
		domain.SourcePrinter:     printerScraper,
		domain.SourceWildberries: wildberriesScraper,
		domain.SourceYandex:      yandexScraperInstance,
		domain.SourceDns:         dnsScraperInstance,
		domain.SourceSprut:       sprutScraperInstance,
		// Новый source: см. internal/parsers/example/doc.go
		// domain.SourceExample: exampleScraper.NewScraper(cfg.Scraping, cfg.Example),
	}

	if discoveryOnly {
		bootstrapDiscoverySeeds(logger, taskRepo, cfg, sources)

		if sourceSelected(sources, domain.SourceDns) {
			dnsFilter := jobSourceFilter(cfg.Jobs, config.JobScrapeDiscovery, domain.SourceDns)
			seedBootstrap, dbBootstrap := dnsFilter.BootstrapMode()
			if seedBootstrap && len(cfg.Dns.DiscoverySeeds) > 0 {
				stats, err := dnsParser.RunDiscoveryBFS(ctx, logger, cfg.Scraping, cfg.Dns, cfg.Dns.DiscoverySeeds, taskRepo)
				if err != nil {
					logger.Error().Err(err).Msg("dns discovery bfs failed")
				} else {
					logger.Info().
						Int("hubs_visited", stats.HubsVisited).
						Int("categories_created", stats.CategoriesCreated).
						Msg("dns discovery bfs complete")
				}
			}

			var phases []string
			if dbBootstrap {
				phases = append(phases, domain.PageTypeDiscovery.String())
			}
			phases = append(phases, domain.PageTypeCategory.String())

			for _, pageType := range phases {
				if err := runScrapePhase(ctx, logger, m, taskRepo, snapshotRepo, sourceToScraper, cfg, []string{domain.SourceDns}, []string{pageType}, true, pageType); err != nil {
					return err
				}
			}
		}

		cleanupWildberriesDiscovery(logger, taskRepo, cfg, sources, cleanupDiscovery)

		if sourceSelected(sources, domain.SourceWildberries) {
			wbFilter := jobSourceFilter(cfg.Jobs, config.JobScrapeDiscovery, domain.SourceWildberries)
			_, dbBootstrap := wbFilter.BootstrapMode()
			if dbBootstrap {
				if err := runScrapePhase(ctx, logger, m, taskRepo, snapshotRepo, sourceToScraper, cfg, []string{domain.SourceWildberries}, nil, true, domain.PageTypeDiscovery.String()); err != nil {
					return err
				}
			}
		}

		logger.Info().Msg("all tasks processed, exiting")
		return nil
	}

	pageTypeFilter := discoveryPageType(pageTypes, discoveryOnly)
	if err := runScrapePhase(ctx, logger, m, taskRepo, snapshotRepo, sourceToScraper, cfg, sources, pageTypes, discoveryOnly, pageTypeFilter); err != nil {
		return err
	}

	logger.Info().Msg("all tasks processed, exiting")
	return nil
}

func runScrapePhase(
	ctx context.Context,
	logger zerolog.Logger,
	m *metrics.Collector,
	taskRepo *repository.TrackedPageRepo,
	snapshotRepo *repository.SnapshotRepo,
	sourceToScraper map[string]scraper.Scraper,
	cfg config.Config,
	sources, pageTypes []string,
	discoveryOnly bool,
	sqlPageType string,
) error {
	sourceFilter, pageTypeFilter := scrapeTaskFilters(sources, pageTypes, discoveryOnly, sqlPageType)

	allTasks, err := taskRepo.GetTasks(sourceFilter, pageTypeFilter, 0)
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	var sourceSet map[string]bool
	if len(sources) > 0 {
		sourceSet = make(map[string]bool, len(sources))
		for _, s := range sources {
			sourceSet[s] = true
		}
	}

	var allowedPageTypes []string
	if sqlPageType != "" {
		allowedPageTypes = []string{sqlPageType}
	} else if discoveryOnly {
		allowedPageTypes = []string{domain.PageTypeDiscovery.String()}
	} else if len(pageTypes) > 0 {
		allowedPageTypes = pageTypes
	}

	var tasks []domain.ScrapeTask
	for _, t := range allTasks {
		if sourceSet != nil && !sourceSet[t.Source] {
			continue
		}
		if len(allowedPageTypes) > 0 && !slices.Contains(allowedPageTypes, t.PageType.String()) {
			continue
		}
		tasks = append(tasks, t)
	}

	scrapeJob := config.JobScrape
	if discoveryOnly {
		scrapeJob = config.JobScrapeDiscovery
	}
	beforeFilter := len(tasks)
	for _, t := range tasks {
		m.AddTasksSelected(ctx, t.Source, t.PageType.String(), scrapeJob, metrics.FilterStageBefore, 1)
	}
	tasks = filterScrapeTasks(tasks, scrapeJob, cfg.Jobs)
	for _, t := range tasks {
		m.AddTasksSelected(ctx, t.Source, t.PageType.String(), scrapeJob, metrics.FilterStageAfter, 1)
	}
	taskByID := make(map[int]domain.ScrapeTask, len(tasks))
	for _, t := range tasks {
		taskByID[t.ID] = t
	}
	logger.Info().
		Str("job", scrapeJob).
		Str("page_type", pageTypeFilter).
		Strs("sources", uniqueTaskSources(tasks)).
		Int("tasks_before_filter", beforeFilter).
		Int("tasks_matched", len(tasks)).
		Msg("scrape tasks after job filters")

	if len(tasks) == 0 {
		logger.Info().Str("page_type", pageTypeFilter).Msg("no active tasks after filtering for phase")
		return nil
	}

	tasksCh := make(chan domain.ScrapeTask)
	resultsCh := make(chan domain.ScrapeResult)
	worker := scraper.NewWorker(logger, sourceToScraper, resultsCh)

	go func() {
		defer close(tasksCh)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case tasksCh <- task:
			}
		}
	}()

	go worker.Run(ctx, tasksCh)

	for result := range resultsCh {
		task := taskByID[result.TrackedPageID]
		pageType := task.PageType.String()
		source := task.Source
		taskLog := withSource(logger, source).With().Str("page_type", pageType).Logger()

		taskLog.Debug().Int("task_id", result.TrackedPageID).Int("resources", len(result.Resources)).Err(result.Err).Msg("run: received result for task")

		if result.Err != nil {
			taskLog.Error().Err(result.Err).Int("task_id", result.TrackedPageID).Msg("scrape error")
			m.AddTaskFinished(ctx, source, pageType, metrics.StatusFailure, 1)
			m.RecordTaskDuration(ctx, source, pageType, result.DurationMs)
			if err := taskRepo.SetStatus(result.TrackedPageID, false, result.DurationMs); err != nil {
				taskLog.Error().Err(err).Msg("update status error")
			}
			continue
		}
		if err := snapshotRepo.SaveResult(result.TrackedPageID, result, result.DurationMs); err != nil {
			taskLog.Error().Err(err).Msg("save snapshot")
			m.AddTaskFinished(ctx, source, pageType, metrics.StatusFailure, 1)
		} else {
			taskLog.Info().Msg("snapshot saved successfully")
			m.AddTaskFinished(ctx, source, pageType, metrics.StatusSuccess, 1)
			if err := taskRepo.SetStatus(result.TrackedPageID, true, result.DurationMs); err != nil {
				taskLog.Error().Err(err).Msg("update status")
			}
		}
		m.RecordTaskDuration(ctx, source, pageType, result.DurationMs)
		taskLog.Debug().Int("task_id", result.TrackedPageID).Msg("run: finished processing task")
	}

	return nil
}

func bootstrapDiscoverySeeds(logger zerolog.Logger, taskRepo *repository.TrackedPageRepo, cfg config.Config, sources []string) {
	if sourceSelected(sources, domain.SourceDns) {
		dnsFilter := jobSourceFilter(cfg.Jobs, config.JobScrapeDiscovery, domain.SourceDns)
		seedBootstrap, _ := dnsFilter.BootstrapMode()
		if seedBootstrap {
			for _, seedURL := range cfg.Dns.DiscoverySeeds {
				if err := taskRepo.CreateTask(domain.SourceDns, domain.PageTypeDiscovery.String(), seedURL); err != nil {
					logger.Error().Err(err).Str("source", domain.SourceDns).Str("url", seedURL).Msg("failed to create DNS discovery seed task")
				}
			}
			for _, query := range cfg.Dns.SearchQueries {
				for page := 1; page <= cfg.Dns.MaxPages; page++ {
					encodedQuery := url.QueryEscape(query)
					searchURL := fmt.Sprintf("https://www.dns-shop.ru/search/?q=%s&page=%d", encodedQuery, page)
					if err := taskRepo.CreateTask(domain.SourceDns, domain.PageTypeDiscovery.String(), searchURL); err != nil {
						logger.Error().Err(err).Str("source", domain.SourceDns).Str("query", query).Int("page", page).Msg("failed to create DNS search discovery task")
					}
				}
			}
		}
	}

	if sourceSelected(sources, domain.SourceWildberries) {
		wbFilter := jobSourceFilter(cfg.Jobs, config.JobScrapeDiscovery, domain.SourceWildberries)
		seedBootstrap, _ := wbFilter.BootstrapMode()
		if seedBootstrap && len(cfg.Wildberries.Discovery.DiscoveryTextQueries) > 0 {
			for _, query := range cfg.Wildberries.Discovery.DiscoveryTextQueries {
				discoveryURL := fmt.Sprintf("wildberries://discovery/%s", query)
				if err := taskRepo.CreateTask(domain.SourceWildberries, domain.PageTypeDiscovery.String(), discoveryURL); err != nil {
					logger.Error().Err(err).Str("source", domain.SourceWildberries).Str("query", query).Msg("failed to create discovery task")
				}
			}
		}
	}
}

func cleanupWildberriesDiscovery(logger zerolog.Logger, taskRepo *repository.TrackedPageRepo, cfg config.Config, sources []string, cleanupDiscovery bool) {
	if !cleanupDiscovery || !sourceSelected(sources, domain.SourceWildberries) {
		return
	}

	allTasks, err := taskRepo.GetTasks("", "", 0)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get tasks for cleanup")
		return
	}

	if len(cfg.Wildberries.Discovery.DiscoveryTextQueries) > 0 {
		queriesMap := make(map[string]bool)
		for _, q := range cfg.Wildberries.Discovery.DiscoveryTextQueries {
			queriesMap[q] = true
		}
		for _, t := range allTasks {
			if t.Source == domain.SourceWildberries && t.PageType == domain.PageTypeDiscovery {
				query := strings.TrimPrefix(t.URL, "wildberries://discovery/")
				if !queriesMap[query] {
					if err := taskRepo.DeleteTaskByID(t.ID); err != nil {
						logger.Error().Err(err).Int("task_id", t.ID).Msg("failed to delete stale discovery task")
					} else {
						logger.Debug().Str("url", t.URL).Msg("deleted stale discovery task")
					}
				}
			}
		}
		return
	}

	for _, t := range allTasks {
		if t.Source == domain.SourceWildberries && t.PageType == domain.PageTypeDiscovery {
			if err := taskRepo.DeleteTaskByID(t.ID); err != nil {
				logger.Error().Err(err).Int("task_id", t.ID).Msg("failed to delete discovery task")
			} else {
				logger.Debug().Str("url", t.URL).Msg("deleted discovery task (no queries in config)")
			}
		}
	}
}

func sourceSelected(sources []string, source string) bool {
	if len(sources) == 0 {
		return true
	}
	return slices.Contains(sources, source)
}

func discoveryPageType(pageTypes []string, discoveryOnly bool) string {
	if discoveryOnly {
		return domain.PageTypeDiscovery.String()
	}
	if len(pageTypes) == 1 {
		return pageTypes[0]
	}
	return ""
}

func scrapeTaskFilters(sources, pageTypes []string, discoveryOnly bool, sqlPageType string) (source, pageType string) {
	if len(sources) == 1 {
		source = sources[0]
	}
	switch {
	case sqlPageType != "":
		pageType = sqlPageType
	case discoveryOnly:
		pageType = domain.PageTypeDiscovery.String()
	case len(pageTypes) == 1:
		pageType = pageTypes[0]
	}
	return source, pageType
}
