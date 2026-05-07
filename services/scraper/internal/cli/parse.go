package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

func parse(ctx context.Context, cfgFile string) error {
	logger := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("service", "scraper").
		Logger()

	logger.Info().Msg("Hello World")

	return nil
}
