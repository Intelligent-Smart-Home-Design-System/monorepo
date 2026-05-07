package temporalx

import (
	"context"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/logging"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
)

type ConnectOptions struct {
	HostPort        string
	Namespace       string
	ConnectAttempts int
	Logger          zerolog.Logger
	Interceptors    []interceptor.ClientInterceptor
}

func Connect(ctx context.Context, options ConnectOptions) (client.Client, error) {
	attempts := options.ConnectAttempts
	if attempts <= 0 {
		attempts = 30
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		temporalClient, err := client.Dial(client.Options{
			HostPort:     options.HostPort,
			Namespace:    options.Namespace,
			Logger:       logging.NewTemporalLogger(options.Logger),
			Interceptors: options.Interceptors,
		})
		if err == nil {
			return temporalClient, nil
		}

		lastErr = err
		options.Logger.Warn().
			Int("attempt", attempt).
			Int("max_attempts", attempts).
			Err(err).
			Msg("Temporal is not ready yet, retrying connection")

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return nil, lastErr
}
