package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/cli"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelsetup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	telemetry := otelsetup.New(ctx, "scraper")
	defer telemetry.Shutdown()

	if err := cli.Execute(telemetry.Log); err != nil {
		os.Exit(1)
	}
}
