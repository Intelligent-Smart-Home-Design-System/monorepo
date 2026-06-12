package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "scraper",
		Short: "Web scraping service for tracking and storing smart home device pages",
	}

	rootCmd.AddCommand(NewRunCmd())

	return rootCmd
}

func Execute() error {
	return NewRootCmd().Execute()
}
