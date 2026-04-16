package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	var cfgFile string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the catalog building job",
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

	logger.Info().Msg("Hello from catalog-builder")

	return nil
}
