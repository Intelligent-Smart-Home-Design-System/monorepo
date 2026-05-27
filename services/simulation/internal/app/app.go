package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/client/ws"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/simulations"
	"golang.org/x/sync/errgroup"
)

const timeoutGraceful = 5 * time.Second

type App struct {
	server *http.Server
}

func New() *App {
	simService := simulations.NewSimulation()
	manager := ws.NewManager(simService)

	mux := http.NewServeMux()
	mux.HandleFunc("/", manager.ServeWS)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	return &App{
		server: server,
	}
}

func (a *App) Run() error {
	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	g, gCtx := errgroup.WithContext(rootCtx)

	g.Go(func() error {
		slog.Info("Starting WebSocket API on :8080")

		err := a.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	})
	g.Go(func() error {
		<-gCtx.Done()

		slog.Debug("stopping server")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			timeoutGraceful,
		)
		defer cancel()

		return a.server.Shutdown(shutdownCtx)
	})

	return g.Wait()
}
