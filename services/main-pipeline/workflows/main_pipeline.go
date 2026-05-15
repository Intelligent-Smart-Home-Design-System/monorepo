package workflows

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func MainPipelineWorkflow(ctx workflow.Context, input pipeline.WorkflowInput) (*pipeline.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	startedAt := workflow.Now(ctx)
	logger.Info("Main pipeline workflow started", "request_id", input.RequestID)

	result := &pipeline.WorkflowResult{
		RequestID: input.RequestID,
		StartedAt: startedAt,
	}

	if input.FloorParser != nil {
		stepResult, err := runFloorParser(ctx, input)
		if err != nil {
			return nil, err
		}
		result.FloorParser = stepResult
	}

	if input.Layout != nil {
		stepResult, err := runLayout(ctx, input)
		if err != nil {
			return nil, err
		}
		result.Layout = stepResult
	}

	if input.DeviceSelection != nil {
		stepResult, err := runDeviceSelection(ctx, input)
		if err != nil {
			return nil, err
		}
		result.DeviceSelection = stepResult
	}

	result.CompletedAt = workflow.Now(ctx)
	logger.Info("Main pipeline workflow completed", "request_id", input.RequestID)
	return result, nil
}

func runFloorParser(ctx workflow.Context, input pipeline.WorkflowInput) (*pipeline.FloorParserActivityOutput, error) {
	logger := workflow.GetLogger(ctx)
	step := input.FloorParser
	logger.Info("Starting floor-parser activity", "source_path", step.SourcePath)

	var output pipeline.FloorParserActivityOutput
	err := executeOnQueue(
		ctx,
		input.TaskQueues.FloorParser,
		input.Activity,
		pipeline.FloorParserActivityName,
		pipeline.FloorParserActivityInput{
			RequestID:  input.RequestID,
			SourcePath: step.SourcePath,
			OutputPath: step.OutputPath,
		},
		&output,
	)
	if err != nil {
		logger.Error("Floor-parser activity failed", "error", err)
		return nil, err
	}

	logger.Info("Floor-parser activity completed", "output_path", output.OutputPath, "walls", output.WallCount)
	return &output, nil
}

func runLayout(ctx workflow.Context, input pipeline.WorkflowInput) (*pipeline.LayoutActivityOutput, error) {
	logger := workflow.GetLogger(ctx)
	step := input.Layout
	logger.Info("Starting layout activity", "apartment_path", step.ApartmentPath)

	var output pipeline.LayoutActivityOutput
	err := executeOnQueue(
		ctx,
		input.TaskQueues.Layout,
		input.Activity,
		pipeline.LayoutActivityName,
		pipeline.LayoutActivityInput{
			RequestID:      input.RequestID,
			ApartmentPath:  step.ApartmentPath,
			OutputPath:     step.OutputPath,
			SelectedLevels: step.SelectedLevels,
		},
		&output,
	)
	if err != nil {
		logger.Error("Layout activity failed", "error", err)
		return nil, err
	}

	logger.Info("Layout activity completed", "output_path", output.OutputPath, "placements", output.PlacementCount)
	return &output, nil
}

func runDeviceSelection(ctx workflow.Context, input pipeline.WorkflowInput) (*pipeline.DeviceSelectionActivityOutput, error) {
	logger := workflow.GetLogger(ctx)
	step := input.DeviceSelection
	logger.Info("Starting device-selection activity", "request_path", step.RequestPath)

	var output pipeline.DeviceSelectionActivityOutput
	err := executeOnQueue(
		ctx,
		input.TaskQueues.DeviceSelection,
		input.Activity,
		pipeline.DeviceSelectionActivityName,
		pipeline.DeviceSelectionActivityInput{
			RequestID:   input.RequestID,
			RequestPath: step.RequestPath,
			OutputPath:  step.OutputPath,
		},
		&output,
	)
	if err != nil {
		logger.Error("Device-selection activity failed", "error", err)
		return nil, err
	}

	logger.Info("Device-selection activity completed", "output_path", output.OutputPath, "solutions", output.SolutionCount)
	return &output, nil
}

func executeOnQueue(ctx workflow.Context, taskQueue string, settings pipeline.ActivitySettings, activityName string, input interface{}, output interface{}) error {
	retryPolicy := &temporal.RetryPolicy{
		MaximumAttempts:    settings.Retry.MaximumAttempts,
		InitialInterval:    settings.Retry.InitialInterval,
		BackoffCoefficient: settings.Retry.BackoffCoefficient,
		MaximumInterval:    settings.Retry.MaximumInterval,
	}

	activityOptions := workflow.ActivityOptions{
		TaskQueue:           taskQueue,
		StartToCloseTimeout: settings.StartToCloseTimeout,
		HeartbeatTimeout:    settings.HeartbeatTimeout,
		RetryPolicy:         retryPolicy,
	}

	stepCtx := workflow.WithActivityOptions(ctx, activityOptions)
	return workflow.ExecuteActivity(stepCtx, activityName, input).Get(stepCtx, output)
}
