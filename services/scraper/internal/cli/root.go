package cli

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func NewRootCmd(log zerolog.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "scraper",
		Short: "Web scraping service for tracking and storing smart home device pages",
	}

	rootCmd.AddCommand(NewScrapeCmd(log))
	rootCmd.AddCommand(NewParseCmd(log))

	return rootCmd
}

func Execute(log zerolog.Logger) error {
	return NewRootCmd(log).Execute()
}
