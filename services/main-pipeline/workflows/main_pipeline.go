package workflows

import (
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	MainPipelineTaskQueue     = "main-pipeline"
	LayoutTaskQueue           = "layout"
	DeviceSelectionTaskQueue  = "device-selection"
	PlaceDevicesActivityName  = "place_devices"
	SelectDevicesActivityName = "select_devices"
)

func MainPipelineWorkflow(ctx workflow.Context, input pipeline.PipelineRequest) (*pipeline.PipelineResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("main pipeline workflow started", "request_id", input.RequestID)

	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	}

	var placed pipeline.LayoutOutput
	layoutCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           LayoutTaskQueue,
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         retryPolicy,
	})
	if err := workflow.ExecuteActivity(layoutCtx, PlaceDevicesActivityName, pipeline.LayoutInput{
		RequestID:      input.RequestID,
		FloorPlan:      input.FloorPlan,
		SelectedLevels: input.SelectedLevels,
	}).Get(ctx, &placed); err != nil {
		return nil, err
	}

	selectionInput, err := pipeline.DeviceSelectionInputFromJSON(input.DeviceSelection)
	if err != nil {
		return nil, err
	}

	var selected pipeline.DeviceSelectionOutput
	selectionCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           DeviceSelectionTaskQueue,
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy:         retryPolicy,
	})
	if err := workflow.ExecuteActivity(selectionCtx, SelectDevicesActivityName, selectionInput).Get(ctx, &selected); err != nil {
		return nil, err
	}

	deviceSelectionResult, err := pipeline.DeviceSelectionOutputToJSON(selected)
	if err != nil {
		return nil, err
	}

	logger.Info("main pipeline workflow completed", "request_id", input.RequestID)
	return &pipeline.PipelineResult{
		RequestID:       input.RequestID,
		ParsedFloorPlan: input.FloorPlan,
		Layout:          placed.Layout,
		DeviceSelection: deviceSelectionResult,
	}, nil
}
