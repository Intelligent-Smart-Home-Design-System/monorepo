package main

import (
	"os"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
