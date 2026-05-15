package activities

import (
	"context"
	"fmt"
	"time"

	"temporal-go-project/internal/logging"
	"temporal-go-project/internal/tracing"
)

type GreetingActivity struct{}

type GreetRequest struct {
	Name string
}

type GreetResponse struct {
	Message   string
	Timestamp time.Time
}

var activityLogger = logging.New("worker-activity")

func (a *GreetingActivity) SayHello(ctx context.Context, req GreetRequest) (*GreetResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "activity.SayHello")
	defer span.End()

	message := fmt.Sprintf("Hello, %s! Welcome to Temporal!", req.Name)
	resp := &GreetResponse{
		Message:   message,
		Timestamp: time.Now(),
	}

	logger := tracing.ContextLogger(ctx, activityLogger)
	logger.Info().
		Str("activity", "SayHello").
		Str("name", req.Name).
		Str("message", resp.Message).
		Msg("Greeting activity completed")

	return resp, nil
}

func (a *GreetingActivity) ProcessData(ctx context.Context, data string) (string, error) {
	ctx, span := tracing.StartSpan(ctx, "activity.ProcessData")
	defer span.End()

	start := time.Now()
	time.Sleep(2 * time.Second)

	result := fmt.Sprintf("Processed: %s (length: %d chars)", data, len(data))

	logger := tracing.ContextLogger(ctx, activityLogger)
	logger.Info().
		Str("activity", "ProcessData").
		Int("input_length", len(data)).
		Dur("duration", time.Since(start)).
		Msg("Data processing activity completed")

	return result, nil
}

func (a *GreetingActivity) SendNotification(ctx context.Context, message string) error {
	ctx, span := tracing.StartSpan(ctx, "activity.SendNotification")
	defer span.End()

	logger := tracing.ContextLogger(ctx, activityLogger)
	logger.Info().
		Str("activity", "SendNotification").
		Str("notification_message", message).
		Msg("Sending notification")

	time.Sleep(1 * time.Second)

	logger.Info().
		Str("activity", "SendNotification").
		Msg("Notification sent")

	return nil
}
