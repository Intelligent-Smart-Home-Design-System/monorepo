package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/client/ws"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/simulations"
	"golang.org/x/sync/errgroup"
)

const timeoutGraceful = 5 * time.Second

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	rootCtx, cancelFunc := context.WithCancel(context.Background()) // graceful shutdown
	g, _ := errgroup.WithContext(rootCtx)

	slog.Debug("Creating components...")

	slog.Debug("Components created")
	slog.Debug("Creating simulations")

	simService := simulations.NewSimulation() // создание симуляций

	slog.Debug("Simulation created")

	setupWebSocketAPI(simService) // запуск API для получения данных о симуляциях

	g.Go(func() error {
		slog.Info("Starting WebSocket API on :8080")
		return http.ListenAndServe(":8080", nil)
	})

	// ===Логика отменты контекста===
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

func setupWebSocketAPI(sim api.SimulationService) {
	manager := ws.NewManager(sim)

	http.HandleFunc("/", manager.ServeWS)
}
