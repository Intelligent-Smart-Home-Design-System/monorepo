package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/cli"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelsetup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	telemetry := otelsetup.New(ctx, "scraper")
	defer telemetry.Shutdown()

	m, err := metrics.New(telemetry.Meter)
	if err != nil {
		telemetry.Log.Warn().Err(err).Msg("failed to initialize scraper metrics")
	}

	if err := cli.Execute(telemetry.Log, m); err != nil {
		os.Exit(1)
	}
}
