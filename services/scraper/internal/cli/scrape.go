package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"slices"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/infra/postgres"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/printer"
	wbScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
	yandexScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/yandex"
)

func NewScrapeCmd() *cobra.Command {
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
			return scrape(ctx, cfgFile, sources, pageTypes, discoveryOnly, cleanupDiscovery)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sources, "sources", nil, "comma-separated list of sources to scrape (e.g., wildberries,sprut)")
	cmd.Flags().StringSliceVar(&pageTypes, "page-types", nil, "comma-separated list of page types (listing, discovery, compatibility)")
	cmd.Flags().BoolVar(&discoveryOnly, "discovery", false, "if true, scrape only discovery pages")
	cmd.Flags().BoolVar(&cleanupDiscovery, "cleanup-discovery", false, "if true, delete discovery tasks that are not in config")

	return cmd
}

func scrape(ctx context.Context, cfgFile string, sources, pageTypes []string, discoveryOnly, cleanupDiscovery bool) error {
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

	taskRepo := repository.NewTrackedPageRepo(db)
	snapshotRepo := repository.NewSnapshotRepo(db, logger)

	printerScraper := printer.NewPrinterScraper()
	wildberriesScraper := wbScraper.NewScraper(
		cfg.Scraping.Timeout,
		cfg.Scraping.Proxy,
		cfg.Scraping.WBCardBasket,
		cfg.Scraping.WBRPS,
		cfg.Scraping.WBSessionPath,
		cfg.Wildberries.Discovery.URLTemplate,
		cfg.Wildberries.Discovery.MaxPages,
	)

	if cfg.Yandex.SupportedZigbeeDevicesURL != "" {
		if err := taskRepo.CreateTask(domain.SourceYandex, domain.PageTypeCompatibility.String(), cfg.Yandex.SupportedZigbeeDevicesURL); err != nil {
			logger.Error().Err(err).Msg("failed to create Yandex compatibility task")
		}
	}

	yandexScraperInstance := yandexScraper.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.Proxy, cfg.Scraping.RateLimitRps)

	sourceToScraper := map[string]scraper.Scraper{
		domain.SourcePrinter:     printerScraper,
		domain.SourceWildberries: wildberriesScraper,
		domain.SourceYandex:      yandexScraperInstance,
	}

	resultsCh := make(chan domain.ScrapeResult)
	worker := scraper.NewWorker(logger, sourceToScraper, resultsCh)

	if len(cfg.Wildberries.Discovery.DiscoveryTextQueries) > 0 {
		// Create tasks for queries from config
		for _, query := range cfg.Wildberries.Discovery.DiscoveryTextQueries {
			discoveryURL := fmt.Sprintf("wildberries://discovery/%s", query)
			if err := taskRepo.CreateTask(domain.SourceWildberries, domain.PageTypeDiscovery.String(), discoveryURL); err != nil {
				logger.Error().Err(err).Str("query", query).Msg("failed to create discovery task")
			}
		}

		if cleanupDiscovery {
			allTasks, err := taskRepo.GetTasks()
			if err != nil {
				logger.Error().Err(err).Msg("failed to get tasks for cleanup")
			} else {
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
			}
		}
	} else if cleanupDiscovery {
		allTasks, err := taskRepo.GetTasks()
		if err != nil {
			logger.Error().Err(err).Msg("failed to get tasks for cleanup")
		} else {
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
	}

	allTasks, err := taskRepo.GetTasks()
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
	if discoveryOnly {
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

	if len(tasks) == 0 {
		logger.Info().Msg("no active tasks after filtering, exiting")
		return nil
	}

	tasksCh := make(chan domain.ScrapeTask)
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
		fmt.Printf("[DEBUG] run: received result for task %d, err=%v, resources=%d\n", result.TrackedPageID, result.Err, len(result.Resources))
		if result.Err != nil {
			logger.Error().Err(result.Err).Int("task_id", result.TrackedPageID).Msg("scrape error")
			if err := taskRepo.SetStatus(result.TrackedPageID, false, result.DurationMs); err != nil {
				logger.Error().Err(err).Msg("update status error")
			}
			continue
		}
		if err := snapshotRepo.SaveResult(result.TrackedPageID, result, result.DurationMs); err != nil {
			logger.Error().Err(err).Msg("save snapshot")
		} else {
			logger.Info().Msg("snapshot saved successfully")
			if err := taskRepo.SetStatus(result.TrackedPageID, true, result.DurationMs); err != nil {
				logger.Error().Err(err).Msg("update status")
			}
		}
		fmt.Printf("[DEBUG] run: finished processing task %d\n", result.TrackedPageID)
	}

	logger.Info().Msg("all tasks processed, exiting")
	return nil
}
