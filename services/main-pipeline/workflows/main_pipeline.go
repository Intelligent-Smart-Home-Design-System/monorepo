package workflows

import (
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	MainPipelineTaskQueue     = "main-pipeline"
	FloorParserTaskQueue      = "floor-parser"
	LayoutTaskQueue           = "layout"
	DeviceSelectionTaskQueue  = "device-selection"
	ParseFloorActivityName    = "parse_floor_json"
	PlaceDevicesActivityName  = "place_devices"
	SelectDevicesActivityName = "select_devices"

	queryPipelineStages = "pipeline_stages"
)

type PipelineStage struct {
	Key     string      `json:"key"`
	Title   string      `json:"title"`
	Status  string      `json:"status"`
	Payload interface{} `json:"payload,omitempty"`
}

type PipelineStagesResult struct {
	WorkflowID string         `json:"workflow_id"`
	RunID      string         `json:"run_id"`
	Status     string         `json:"status"`
	Progress   float64        `json:"progress"`
	Stages     []PipelineStage `json:"stages"`
}

func MainPipelineWorkflow(ctx workflow.Context, input pipeline.PipelineRequest) (*pipeline.PipelineResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("main pipeline workflow started", "request_id", input.RequestID)

	stages := []PipelineStage{
		{Key: ParseFloorActivityName, Title: "Парсинг плана", Status: "pending"},
		{Key: PlaceDevicesActivityName, Title: "Расстановка", Status: "pending"},
		{Key: SelectDevicesActivityName, Title: "Подбор устройств", Status: "pending"},
	}

	if err := workflow.SetQueryHandler(ctx, queryPipelineStages, func() ([]PipelineStage, error) {
		return stages, nil
	}); err != nil {
		return nil, err
	}

	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	}

	stages[0].Status = "running"

	var parsed pipeline.FloorParserOutput
	parseCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           FloorParserTaskQueue,
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         retryPolicy,
	})
	if err := workflow.ExecuteActivity(parseCtx, ParseFloorActivityName, pipeline.FloorParserInput{
		RequestID: input.RequestID,
		FloorPlan: input.FloorPlan,
	}).Get(ctx, &parsed); err != nil {
		stages[0].Status = "failed"
		return nil, err
	}
	stages[0].Status = "completed"
	stages[0].Payload = parsed.FloorPlan

	stages[1].Status = "running"

	var placed pipeline.LayoutOutput
	layoutCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           LayoutTaskQueue,
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         retryPolicy,
	})
	if err := workflow.ExecuteActivity(layoutCtx, PlaceDevicesActivityName, pipeline.LayoutInput{
		RequestID:      input.RequestID,
		FloorPlan:      parsed.FloorPlan,
		SelectedLevels: input.SelectedLevels,
	}).Get(ctx, &placed); err != nil {
		stages[1].Status = "failed"
		return nil, err
	}
	stages[1].Status = "completed"
	stages[1].Payload = placed.Layout

	selectionInput, err := pipeline.DeviceSelectionInputFromJSON(input.DeviceSelection)
	if err != nil {
		return nil, err
	}

	stages[2].Status = "running"

	var selected pipeline.DeviceSelectionOutput
	selectionCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           DeviceSelectionTaskQueue,
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy:         retryPolicy,
	})
	if err := workflow.ExecuteActivity(selectionCtx, SelectDevicesActivityName, selectionInput).Get(ctx, &selected); err != nil {
		stages[2].Status = "failed"
		return nil, err
	}
	stages[2].Status = "completed"

	deviceSelectionResult, err := pipeline.DeviceSelectionOutputToJSON(selected)
	if err != nil {
		return nil, err
	}
	stages[2].Payload = deviceSelectionResult

	logger.Info("main pipeline workflow completed", "request_id", input.RequestID)
	return &pipeline.PipelineResult{
		RequestID:       input.RequestID,
		ParsedFloorPlan: parsed.FloorPlan,
		Layout:          placed.Layout,
		DeviceSelection: deviceSelectionResult,
	}, nil
}
