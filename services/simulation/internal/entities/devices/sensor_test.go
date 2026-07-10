package devices

import "testing"

// TestRadiusSensorInput_RejectsMissingTarget проверяет, что некорректный move payload возвращает ошибку вместо panic.
func TestRadiusSensorInput_RejectsMissingTarget(t *testing.T) {
	if _, _, err := radiusSensorInput([]byte(`{"kind":"human:move"}`), 0, 0, 1); err == nil {
		t.Fatal("expected missing to error")
	}
}

// TestRadiusSensorInput_HandlesIncidentBlocks проверяет пересечение sensor radius с polygon incident-блока.
func TestRadiusSensorInput_HandlesIncidentBlocks(t *testing.T) {
	kind, active, err := radiusSensorInput([]byte(`{
		"kind":"smoke:spread",
		"blocks":[{"x":1,"y":1,"size":1,"points":[[0.5,0.5],[1.5,0.5],[1.5,1.5],[0.5,1.5]]}]
	}`), 1, 1, 0.2)
	if err != nil {
		t.Fatalf("parse incident payload: %v", err)
	}
	if kind != "smoke:spread" || !active {
		t.Fatalf("unexpected incident result: kind=%q active=%v", kind, active)
	}
}

// TestRadiusSensorInput_RecognizesBlocksStructurally проверяет incident payload даже при неизвестном kind.
func TestRadiusSensorInput_RecognizesBlocksStructurally(t *testing.T) {
	_, active, err := radiusSensorInput([]byte(`{
		"kind":"incident:spread",
		"blocks":[{"x":1,"y":1,"size":1,"points":[[0.5,0.5],[1.5,0.5],[1.5,1.5],[0.5,1.5]]}]
	}`), 1, 1, 0.2)
	if err != nil || !active {
		t.Fatalf("structural incident payload was not handled: active=%v err=%v", active, err)
	}
}
