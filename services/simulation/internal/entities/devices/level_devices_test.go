package devices

import "testing"

func TestLevelDevicesKeepPowerAndLevelIndependent(t *testing.T) {
	t.Run("smart lamp", func(t *testing.T) {
		device := &SmartLamp{}
		assertIndependentLevelState(
			t,
			func(turnOn *bool, level *int) (*bool, *int) {
				out := device.HandleEvent(SmartLampData{TurnOn: turnOn, Percents: level})
				return out.TurnOn, out.Percents
			},
		)
	})

	t.Run("smart dimmer", func(t *testing.T) {
		device := &SmartDimmer{}
		assertIndependentLevelState(
			t,
			func(turnOn *bool, level *int) (*bool, *int) {
				out := device.HandleEvent(DimmerData{TurnOn: turnOn, Percents: level})
				return out.TurnOn, out.Percents
			},
		)
	})

	t.Run("smart curtains", func(t *testing.T) {
		device := &SmartCurtains{}
		assertIndependentLevelState(
			t,
			func(turnOn *bool, level *int) (*bool, *int) {
				out := device.HandleEvent(CurtainsData{TurnOn: turnOn, Percents: level})
				return out.TurnOn, out.Percents
			},
		)
	})
}

func assertIndependentLevelState(
	t *testing.T,
	handle func(turnOn *bool, level *int) (*bool, *int),
) {
	t.Helper()

	level := 65
	turnOn, currentLevel := handle(nil, &level)
	if turnOn == nil || *turnOn {
		t.Fatalf("level update changed power state: %v", turnOn)
	}
	if currentLevel == nil || *currentLevel != level {
		t.Fatalf("level = %v, want %d", currentLevel, level)
	}

	on := true
	turnOn, currentLevel = handle(&on, nil)
	if turnOn == nil || !*turnOn {
		t.Fatalf("power update was not applied: %v", turnOn)
	}
	if currentLevel == nil || *currentLevel != level {
		t.Fatalf("power update changed level: %v, want %d", currentLevel, level)
	}
}
