package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/printer"
	sprutPkg "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/sprut"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/worker"
)

func NewRunCmd() *cobra.Command {
	var cfgFile string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the scraping job",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			return run(ctx, cfgFile)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")

	return cmd
}

func run(ctx context.Context, cfgFile string) error {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	var cfg config.Config
	readConfig(cfgFile, &cfg)

	logger.Info().Msgf("rate limit from config: %f", cfg.Scraping.RateLimitRps)

	// TODO: get tasks from db
	//tasksCh := getTasks()

	db, err := connectDB(cfg.Database)
    if err != nil {
        return fmt.Errorf("connect to db: %w", err)
    }
    defer db.Close()

	taskRepo := repository.NewTrackedPageRepo(db)
    snapshotRepo := repository.NewSnapshotRepo(db)

	printer := printer.NewPrinterScraper()
	sprutScraper := sprutScraper.NewScraper(cfg.Scraping.Timeout)

	sourceToScraper := map[string]worker.Scraper{
		"printer": printer,
		"sprut":   sprutScraper
	}

	resultsCh := make(chan domain.ScrapeResult)

	worker := worker.NewWorker(logger, sourceToScraper, resultsCh)

	tasksCh := make(chan domain.ScrapeTask)
    go func() {
        defer close(tasksCh)
        for {
            select {
            case <-ctx.Done():
                return
            default:
                tasks, err := taskRepo.GetTasks()
                if err != nil {
                    logger.Sugar().Errorf("get tasks: %v", err)
                    time.Sleep(5 * time.Second)
                    continue
                }
                if len(tasks) == 0 {
                    time.Sleep(5 * time.Second)
                    continue
                }
                for _, task := range tasks {
                    tasksCh <- task
                }
            }
        }
    }()

	go worker.Run(ctx, tasksCh)

	for result := range resultsCh {
<<<<<<< HEAD
		for _, resource := range result.Resources {
			logger.Info().Msgf("scraped %s: %s", resource.Name, string(resource.ResponseBody))
=======
		if err := snapshotRepo.SaveResult(result.TrackedPageID, result, result.DurationMs); err != nil {
			logger.Sugar().Errorf("save snapshot: %v", err)
		} else {
			if err := taskRepo.SetStatus(result.TrackedPageID, true, result.DurationMs); err != nil {
				logger.Sugar().Errorf("update status: %v", err)
			}
>>>>>>> 1a79677 (Updates for db)
		}
    }

	return nil
}

func getTasks() <-chan domain.ScrapeTask {
	tasks := []domain.ScrapeTask{
		{
			Source:   "printer",
			PageType: "none",
			URL:      "http://www.example.com",
		},
		{
        Source:   "sprut",
        PageType: "article",
        URL:      "https://sprut.ai/catalog",
    },
	}

	tasksCh := make(chan domain.ScrapeTask)

	go func() {
		for _, task := range tasks {
			tasksCh <- task
		}
		close(tasksCh)
	}()

	return tasksCh
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

func connectDB(cfg config.DatabaseConfig) (*sql.DB, error) {
    connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    if err := db.Ping(); err != nil {
        return nil, err
    }
    return db, nil
}
