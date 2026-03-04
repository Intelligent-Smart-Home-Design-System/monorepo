package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/extractor/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/extractor/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/extractor/internal/scrapers/printer"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/extractor/internal/worker"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
	logger, _ := zap.NewDevelopment()

	var cfg config.Config
	readConfig(cfgFile, &cfg)

	logger.Sugar().Infof("rate limit from config: %f", cfg.Scraping.RateLimitRps)

	// TODO: get tasks from db
	tasksCh := getTasks()

	printer := printer.NewPrinterScraper()

	sourceToScraper := map[string]worker.Scraper{
		"printer": printer,
		// TODO: "sprut_ai": sprutScraper,
		// "wildberries": wildberriesScraper
	}

	resultsCh := make(chan domain.ScrapeResult)

	worker := worker.NewWorker(logger, sourceToScraper, resultsCh)

	go worker.Run(ctx, tasksCh)

	// TODO: save results to db
	for result := range resultsCh {
		for _, resource := range result.Resources {
			logger.Sugar().Infof("scraped %s: %s", resource.Name, string(resource.ResponseBody))
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
			Source:   "printer",
			PageType: "none",
			URL:      "http://www.example.com",
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
