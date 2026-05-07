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
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/printer"
	wbScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/worker"
)

func NewScrapeCmd() *cobra.Command {
	var cfgFile string
	var sources []string

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Run the scraping job",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			return scrape(ctx, cfgFile, sources)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")
	cmd.Flags().StringSliceVar(&sources, "sources", nil,
		fmt.Sprintf("comma-separated list of sources to scrape (available: %s, %s); defaults to all",
			domain.SourcePrinter, domain.SourceWildberries))

	return cmd
}

func scrape(ctx context.Context, cfgFile string, sources []string) error {
	logger := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("service", "scraper").
		Logger()

	var cfg config.Config
	if err := readConfig(cfgFile, &cfg); err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}

	logger.Info().Msgf("rate limit from config: %f", cfg.Scraping.RateLimitRps)

	db, err := postgres.NewDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer db.Close()

	taskRepo := repository.NewTrackedPageRepo(db)
	snapshotRepo := repository.NewSnapshotRepo(db)

	printerScraper := printer.NewPrinterScraper()
	// sprutScraper := sprutPkg.NewScraper(cfg.Scraping.Timeout, cfg.Scraping.UserAgent)
	wildberriesScraper := wbScraper.NewScraper(
		cfg.Scraping.Timeout,
		cfg.Scraping.Proxy,
		cfg.Scraping.WBCardBasket,
		cfg.Scraping.WBRPS,
		cfg.Scraping.WBSessionPath,
	)

	allScrapers := map[string]worker.Scraper{
		domain.SourcePrinter:     printerScraper,
		domain.SourceWildberries: wildberriesScraper,
	}

	// Filter scrapers by --sources flag; use all if not specified.
	sourceToScraper := allScrapers
	if len(sources) > 0 {
		sourceToScraper = make(map[string]worker.Scraper, len(sources))
		for _, s := range sources {
			scraper, ok := allScrapers[s]
			if !ok {
				return fmt.Errorf("unknown source %q (available: %s, %s)",
					s, domain.SourcePrinter, domain.SourceWildberries)
			}
			sourceToScraper[s] = scraper
		}
		logger.Info().Strs("sources", sources).Msg("running with filtered sources")
	}

	resultsCh := make(chan domain.ScrapeResult)
	w := worker.NewWorker(logger, sourceToScraper, resultsCh)

	tasks, err := taskRepo.GetTasks()
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}
	if len(tasks) == 0 {
		logger.Info().Msg("no active tasks, exiting")
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

	go w.Run(ctx, tasksCh)

	for result := range resultsCh {
		if result.Err != nil {
			logger.Error().
				Err(result.Err).
				Int("task_id", result.TrackedPageID).
				Msg("scrape error")
			if err := taskRepo.SetStatus(result.TrackedPageID, false, result.DurationMs); err != nil {
				logger.Error().Err(err).Int("task_id", result.TrackedPageID).Msg("update status error")
			}
			continue
		}
		if err := snapshotRepo.SaveResult(result.TrackedPageID, result, result.DurationMs); err != nil {
			logger.Error().Err(err).Int("task_id", result.TrackedPageID).Msg("save snapshot error")
		} else {
			logger.Info().Int("task_id", result.TrackedPageID).Msg("snapshot saved successfully")
			if err := taskRepo.SetStatus(result.TrackedPageID, true, result.DurationMs); err != nil {
				logger.Error().Err(err).Int("task_id", result.TrackedPageID).Msg("update status error")
			}
		}
	}

	logger.Info().Msg("all tasks processed, exiting")
	return nil
}

func readConfig(cfgFile string, cfg *config.Config) error {
	viper.SetConfigFile(cfgFile)

	viper.SetEnvPrefix("SCRAPER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config: %w", err)
		}
		fmt.Fprintln(os.Stderr, "no config file found, using defaults and environment variables")
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	return nil
}
