package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/infra/postgres"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/printer"
	wbScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
)

func NewScrapeCmd() *cobra.Command {
	var cfgFile string
	var sources []string

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Scrape pages from tracked tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()
			return scrape(ctx, cfgFile, sources)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sources, "sources", nil, "comma-separated list of sources to scrape (e.g., wildberries,sprut)")

	return cmd
}

func scrape(ctx context.Context, cfgFile string, sources []string) error {
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
	// sprutScraper := sprutPkg.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.UserAgent)
	wildberriesScraper := wbScraper.NewScraper(
		cfg.Scraping.Timeout,
		cfg.Scraping.Proxy,
		cfg.Scraping.WBCardBasket,
		cfg.Scraping.WBRPS,
		cfg.Scraping.WBSessionPath,
	)

	sourceToScraper := map[string]scraper.Scraper{
		"printer": printerScraper,
		// "sprut":      sprutScraper,
		"wildberries": wildberriesScraper,
	}

	resultsCh := make(chan domain.ScrapeResult)

	worker := scraper.NewWorker(logger, sourceToScraper, resultsCh)

	allTasks, err := taskRepo.GetTasks()
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	// Filter tasks by sources if provided
	var tasks []domain.ScrapeTask
	if len(sources) > 0 {
		sourceSet := make(map[string]bool, len(sources))
		for _, s := range sources {
			sourceSet[s] = true
		}
		for _, t := range allTasks {
			if sourceSet[t.Source] {
				tasks = append(tasks, t)
			}
		}
		if len(tasks) == 0 {
			logger.Info().Msgf("no active tasks for sources: %v", sources)
			return nil
		}
	} else {
		tasks = allTasks
		if len(tasks) == 0 {
			logger.Info().Msg("no active tasks, exiting")
			return nil
		}
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

func readConfig(cfgFile string, cfg *config.Config) error {
	viper.SetConfigFile(cfgFile)

	// Environment variable binding
	viper.SetEnvPrefix("SCRAPER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config: %w", err)
		}
		// Config file not found; use defaults + env vars
		fmt.Fprintln(os.Stderr, "No config file found, using defaults and environment variables")
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	return nil
}
