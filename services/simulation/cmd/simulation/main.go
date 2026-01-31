package main

import (
	"context"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/fetcher"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/sender"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/simulation"
)

func main() {
	// TODO: setup logger

	rootCtx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// TODO: setup all components (fetcher, sender, simulation, ...)
	simFetcher := fetcher.NewSimFetcher()
	simSender := sender.NewSimSender()

	sim := simulation.NewSimulation(simFetcher, simSender)
	err := sim.Run(rootCtx)
	if err != nil {
		slog.Error("cannot initialize simulation")
		return
	}

	// TODO: create simulation and start
}
