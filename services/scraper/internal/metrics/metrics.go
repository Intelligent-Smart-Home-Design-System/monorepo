package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	FilterStageBefore = "before"
	FilterStageAfter  = "after"

	OutcomeMatched           = "matched"
	OutcomeParsed            = "parsed"
	OutcomeSaved             = "saved"
	OutcomeSkippedNoMarkers  = "skipped_no_markers"
	OutcomeParseError        = "parse_error"
	OutcomeSaveError         = "save_error"

	StatusSuccess = "success"
	StatusFailure = "failure"
)

type Collector struct {
	tasksSelected   metric.Int64Counter
	tasksFinished   metric.Int64Counter
	taskDuration    metric.Float64Histogram
	parseSnapshots  metric.Int64Counter
}

func New(meter metric.Meter) (*Collector, error) {
	tasksSelected, err := meter.Int64Counter(
		"scraper_tasks_selected_total",
		metric.WithDescription("Scrape tasks selected before or after job filters"),
	)
	if err != nil {
		return nil, err
	}

	tasksFinished, err := meter.Int64Counter(
		"scraper_tasks_finished_total",
		metric.WithDescription("Scrape tasks finished grouped by status"),
	)
	if err != nil {
		return nil, err
	}

	taskDuration, err := meter.Float64Histogram(
		"scraper_task_duration_seconds",
		metric.WithDescription("Scrape task duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	parseSnapshots, err := meter.Int64Counter(
		"scraper_parse_snapshots_total",
		metric.WithDescription("Parse pipeline snapshot outcomes"),
	)
	if err != nil {
		return nil, err
	}

	return &Collector{
		tasksSelected:  tasksSelected,
		tasksFinished:  tasksFinished,
		taskDuration:   taskDuration,
		parseSnapshots: parseSnapshots,
	}, nil
}

func (c *Collector) AddTasksSelected(ctx context.Context, source, pageType, job, filterStage string, delta int64) {
	if c == nil || delta == 0 {
		return
	}
	c.tasksSelected.Add(ctx, delta, metric.WithAttributes(
		attribute.String("source", source),
		attribute.String("page_type", pageType),
		attribute.String("job", job),
		attribute.String("filter_stage", filterStage),
	))
}

func (c *Collector) AddTaskFinished(ctx context.Context, source, pageType, status string, delta int64) {
	if c == nil || delta == 0 {
		return
	}
	c.tasksFinished.Add(ctx, delta, metric.WithAttributes(
		attribute.String("source", source),
		attribute.String("page_type", pageType),
		attribute.String("status", status),
	))
}

func (c *Collector) RecordTaskDuration(ctx context.Context, source, pageType string, durationMs int) {
	if c == nil || durationMs < 0 {
		return
	}
	c.taskDuration.Record(ctx, float64(durationMs)/1000, metric.WithAttributes(
		attribute.String("source", source),
		attribute.String("page_type", pageType),
	))
}

func (c *Collector) AddParseSnapshots(ctx context.Context, source, pageType, job, outcome, filterStage string, delta int64) {
	if c == nil || delta == 0 {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("source", source),
		attribute.String("page_type", pageType),
		attribute.String("job", job),
		attribute.String("outcome", outcome),
	}
	if filterStage != "" {
		attrs = append(attrs, attribute.String("filter_stage", filterStage))
	}
	c.parseSnapshots.Add(ctx, delta, metric.WithAttributes(attrs...))
}
