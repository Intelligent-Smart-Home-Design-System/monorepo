package parser

import "context"

// ParseMetrics records per-snapshot parse outcomes (optional instrumentation).
type ParseMetrics interface {
	AddParseSnapshots(ctx context.Context, source, pageType, job, outcome, filterStage string, delta int64)
}
