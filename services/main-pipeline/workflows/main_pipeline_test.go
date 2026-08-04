package workflows

import (
	"context"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// TestMainPipelineWorkflowPreservesLayoutDependencies verifies that scenarios
// returned by layout-worker reach the public pipeline result unchanged.
func TestMainPipelineWorkflowPreservesLayoutDependencies(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	dependencies := map[string][]string{
		"switch-1": {"curtains-1", "dimmer-1"},
	}
	env.RegisterActivityWithOptions(
		func(context.Context, pipeline.LayoutInput) (pipeline.LayoutOutput, error) {
			return pipeline.LayoutOutput{
				Layout:       map[string]interface{}{"placements": map[string]interface{}{}},
				Dependencies: dependencies,
			}, nil
		},
		activity.RegisterOptions{Name: PlaceDevicesActivityName},
	)
	env.RegisterActivityWithOptions(
		func(context.Context, pipeline.DeviceSelectionInput) (pipeline.DeviceSelectionOutput, error) {
			return pipeline.DeviceSelectionOutput{}, nil
		},
		activity.RegisterOptions{Name: SelectDevicesActivityName},
	)

	env.ExecuteWorkflow(MainPipelineWorkflow, pipeline.PipelineRequest{
		RequestID:       "dependencies-test",
		FloorPlan:       map[string]interface{}{"rooms": []interface{}{}},
		SelectedLevels:  map[string]string{},
		DeviceSelection: map[string]interface{}{},
	})

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}

	var result pipeline.PipelineResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("GetWorkflowResult() error = %v", err)
	}
	if got := result.Dependencies["switch-1"]; len(got) != 2 || got[0] != "curtains-1" || got[1] != "dimmer-1" {
		t.Fatalf("Dependencies = %#v, want %#v", result.Dependencies, dependencies)
	}
}
