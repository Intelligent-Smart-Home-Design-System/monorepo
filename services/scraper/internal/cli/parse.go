package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/wildberries"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func NewParseCmd() *cobra.Command {
	var cfgFile string

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Run the parsing job",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			return parse(ctx, cfgFile)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")

	return cmd
}

type Parser interface {
}

func parse(ctx context.Context, cfgFile string) error {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	var cfg config.Config
	readConfig(cfgFile, &cfg)

	logger.Info().Msgf("rate limit from config: %f", cfg.Scraping.RateLimitRps)

	db, err := connectDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}

	listingParsers := []parser.SourceParser[*domain.ListingParseResult]{
		wildberries.NewListingParser(),
	}

	repo := repository.NewSnapshotRepo(db, logger)

	worker := parser.NewWorker(logger, domain.PageTypeListing, repo, listingParsers)

	listings := worker.Parse(ctx)

	logger.Debug().Msgf("parsed %d listings", len(listings))

	for _, listing := range listings {
		if err := repo.SaveListingParseResult(listing); err != nil {
			logger.Error().Err(err).Msg("failed to save parse result")
		}
	}

	return nil
}
