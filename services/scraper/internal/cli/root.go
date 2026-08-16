package cli

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func NewRootCmd(log zerolog.Logger, m *metrics.Collector) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "scraper",
		Short: "Web scraping service for tracking and storing smart home device pages",
	}

	rootCmd.AddCommand(NewScrapeCmd(log, m))
	rootCmd.AddCommand(NewParseCmd(log, m))

	return rootCmd
}

func Execute(log zerolog.Logger, m *metrics.Collector) error {
	return NewRootCmd(log, m).Execute()
}
