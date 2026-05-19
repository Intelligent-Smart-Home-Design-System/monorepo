package workflows

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/pipeline"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func CatalogPipelineWorkflow(ctx workflow.Context, input pipeline.WorkflowInput) (*pipeline.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	startedAt := workflow.Now(ctx)
	logger.Info("Catalog pipeline workflow started", "jobs", len(input.Jobs))

	retryPolicy := &temporal.RetryPolicy{
		MaximumAttempts:    input.Activity.Retry.MaximumAttempts,
		InitialInterval:    input.Activity.Retry.InitialInterval,
		BackoffCoefficient: input.Activity.Retry.BackoffCoefficient,
		MaximumInterval:    input.Activity.Retry.MaximumInterval,
	}

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: input.Activity.StartToCloseTimeout,
		HeartbeatTimeout:    input.Activity.HeartbeatTimeout,
		RetryPolicy:         retryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	results := make([]pipeline.RunContainerResult, 0, len(input.Jobs))
	for _, job := range input.Jobs {
		params := pipeline.RunContainerParams{
			Name:       job.Name,
			Image:      job.Image,
			Command:    job.Command,
			ConfigPath: job.ConfigPath,
			EnvMapping: job.EnvMapping,
		}

		logger.Info("Starting pipeline job", "job", job.Name, "image", job.Image)

		var result pipeline.RunContainerResult
		err := workflow.ExecuteActivity(ctx, "RunContainer", params).Get(ctx, &result)
		if err != nil {
			logger.Error("Pipeline job failed", "job", job.Name, "error", err)
			return nil, err
		}

		results = append(results, result)
		logger.Info("Pipeline job completed", "job", job.Name, "exit_code", result.ExitCode)
	}

	workflowResult := &pipeline.WorkflowResult{
		StartedAt:   startedAt,
		CompletedAt: workflow.Now(ctx),
		Jobs:        results,
	}

	logger.Info("Catalog pipeline workflow completed successfully")
	return workflowResult, nil
}
