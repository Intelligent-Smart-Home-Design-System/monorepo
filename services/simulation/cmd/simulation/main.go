package main

import (
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/app"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/logging"
)

func main() {
	logging.Setup("simulation")

	slog.Info("starting simulations service")

	simApp := app.New()
	if err := simApp.Run(); err != nil {
		slog.Error("application stopped with error", "error", err)
	}
}
