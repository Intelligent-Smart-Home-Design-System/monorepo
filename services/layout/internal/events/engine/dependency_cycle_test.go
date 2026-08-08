package engine

import (
	"reflect"
	"testing"
)

// TestFindDependencyCycle verifies cycle detection and the returned closed chain.
func TestFindDependencyCycle(t *testing.T) {
	dependencies := map[string][]string{
		"sensor": {"lamp"},
		"lamp":   {"switch"},
		"switch": {"sensor"},
	}

	want := []string{"lamp", "switch", "sensor", "lamp"}
	if got := FindDependencyCycle(dependencies); !reflect.DeepEqual(got, want) {
		t.Fatalf("FindDependencyCycle() = %v, want %v", got, want)
	}
}

// TestFindDependencyCycleReturnsNil verifies that an acyclic dependency graph is accepted.
func TestFindDependencyCycleReturnsNil(t *testing.T) {
	dependencies := map[string][]string{
		"sensor": {"lamp"},
		"lamp":   nil,
	}

	if got := FindDependencyCycle(dependencies); got != nil {
		t.Fatalf("FindDependencyCycle() = %v, want nil", got)
	}
}
