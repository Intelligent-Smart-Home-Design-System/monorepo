package workflows

import (
	"fmt"
	"time"
	
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"temporal-go-project/activities"
)

type GreetingWorkflowInput struct {
	Name string
	Data string
}

type GreetingWorkflowResult struct {
	GreetingMessage string
	ProcessedData   string
	CompletedAt     time.Time
}

func GreetingWorkflow(ctx workflow.Context, input GreetingWorkflowInput) (*GreetingWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("GreetingWorkflow started", "Name", input.Name)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var greetActivity activities.GreetingActivity
	var greetResp activities.GreetResponse
	
	err := workflow.ExecuteActivity(ctx, greetActivity.SayHello, activities.GreetRequest{
		Name: input.Name,
	}).Get(ctx, &greetResp)
	
	if err != nil {
		logger.Error("SayHello activity failed", "Error", err)
		return nil, err
	}
	
	logger.Info("Greeting received", "Message", greetResp.Message)

	var processedData string
	err = workflow.ExecuteActivity(ctx, greetActivity.ProcessData, input.Data).Get(ctx, &processedData)
	
	if err != nil {
		logger.Error("ProcessData activity failed", "Error", err)
		return nil, err
	}
	
	logger.Info("Data processed", "Result", processedData)

	notificationMessage := fmt.Sprintf("Workflow завершен для %s", input.Name)
	err = workflow.ExecuteActivity(ctx, greetActivity.SendNotification, notificationMessage).Get(ctx, nil)
	
	if err != nil {
		logger.Error("SendNotification activity failed", "Error", err)
		return nil, err
	}

	result := &GreetingWorkflowResult{
		GreetingMessage: greetResp.Message,
		ProcessedData:   processedData,
		CompletedAt:     workflow.Now(ctx),
	}

	logger.Info("GreetingWorkflow completed successfully")
	return result, nil
}

func SimpleWorkflow(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("SimpleWorkflow started", "Name", name)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var greetActivity activities.GreetingActivity
	var greetResp activities.GreetResponse
	
	err := workflow.ExecuteActivity(ctx, greetActivity.SayHello, activities.GreetRequest{
		Name: name,
	}).Get(ctx, &greetResp)
	
	if err != nil {
		return "", err
	}

	return greetResp.Message, nil
}
