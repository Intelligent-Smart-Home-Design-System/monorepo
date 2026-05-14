package main

import (
	"log/slog"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/app"
)

const timeoutGraceful = 5 * time.Second

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Info("Starting simulations service")

	simApp := app.New()
	if err := simApp.Run(); err != nil {
		slog.Error("Application stopped with error", "error", err)
	}
}
