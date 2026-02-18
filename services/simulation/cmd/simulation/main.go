package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/fetcher"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/sender"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/simulations"
	"golang.org/x/sync/errgroup"
)

const timeoutGraceful = 5 * time.Second

func main() {
	// TODO: setup logger
	slog.SetLogLoggerLevel(slog.LevelDebug)

	rootCtx, cancelFunc := context.WithCancel(context.Background()) // graceful shutdown
	g, gCtx := errgroup.WithContext(rootCtx)

	slog.Debug("Creating components...")

	simFetcher := fetcher.NewSimFetcher()
	simSender := sender.NewSimSender()

	slog.Debug("Components created")
	slog.Debug("Creating simulations")

	sim := simulations.NewSimulation(simFetcher, simSender) // создание симуляций

	slog.Debug("Simulation created")
	slog.Debug("Starting simulations...")

	g.Go(func() error { // запуск сервиса симуляций
		return sim.Run(gCtx)
	})

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	done := make(chan error, 1)

	go func() {
		done <- g.Wait()
	}()

	select {
	case sig := <-stopCh:
		slog.Info("Received signal, stopping simulations service...", "signal", sig.String())
		cancelFunc()

		select {
		case err := <-done:
			switch {
			case errors.Is(err, context.Canceled):
				slog.Info("Context cancelled, simulations stopped", "error", err)
			default:
				slog.Error("Error while running simulations", "error", err)
			}
		case <-time.After(timeoutGraceful):
			slog.Warn("Graceful timeout is over, simulations stopped")
		}
	case err := <-done:
		if err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				slog.Info("Context cancelled, simulations stopped", "error", err)
			default:
				slog.Error("Error while running simulations", "error", err)
			}
		} else {
			slog.Info("simulations stopped")
		}
	}
}
