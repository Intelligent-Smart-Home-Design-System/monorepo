package engine

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/stretchr/testify/assert"
)

func TestMakeScenarioDependencies(t *testing.T) {
    layout := &apartment.Layout{
        Placements: map[string][]*device.Placement{
            "living_room_id": {
                {
                    Device: &device.Device{
                        ID:   "window_sensor_id",
                        Type: "window_sensor",
                    },
                },
                {
                    Device: &device.Device{
                        ID:   "camera1_id",
                        Type: "camera",
                    },
                },
                {
                    Device: &device.Device{
                        ID:   "tv_id",
                        Type: "smart_tv",
                    },
                },
            },
            "hall_room_id": {
                {
                    Device: &device.Device{
                        ID:   "smart_lock_id",
                        Type: "smart_lock",
                    },
                },
                {
                    Device: &device.Device{
                        ID:   "smart_doorbell_id",
                        Type: "smart_doorbell",
                    },
                },
                {
                    Device: &device.Device{
                        ID:   "door_sensor_id",
                        Type: "door_sensor",
                    },
                },
				{
                    Device: &device.Device{
                        ID:   "camera2_id",
                        Type: "camera",
                    },
                },
            },
            "kitchen_id": {
                {
                    Device: &device.Device{
                        ID:   "water_leak_sensor_id",
                        Type: "water_leak_sensor",
                    },
                },
            },
			"passage_id": {
				{
					Device: &device.Device{
                        ID:   "smart_siren_id",
                        Type: "smart_siren",
                    },
				},
			},
        },
    }

    engine := &Engine{}
    dependencies, err := engine.MakeScenarioDependencies(layout)

    assert.NotNil(t, dependencies)
	assert.NoError(t, err)

	assert.Contains(t, dependencies["window_sensor_id"], "smart_siren_id")
	assert.Contains(t, dependencies["window_sensor_id"], "camera1_id")
	assert.Len(t, dependencies["window_sensor_id"], 2)

	assert.Contains(t, dependencies["camera1_id"], "smart_siren_id")
	assert.Len(t, dependencies["camera1_id"], 1)

	assert.Len(t, dependencies["tv_id"], 0)

	assert.Contains(t, dependencies["smart_lock_id"], "camera2_id")
	assert.Len(t, dependencies["smart_lock_id"], 1)

	assert.Contains(t, dependencies["smart_doorbell_id"], "smart_lock_id")
	assert.Contains(t, dependencies["smart_doorbell_id"], "camera2_id")
	assert.Len(t, dependencies["smart_doorbell_id"], 2)

	assert.Contains(t, dependencies["door_sensor_id"], "smart_lock_id")
	assert.Contains(t, dependencies["door_sensor_id"], "camera2_id")
	assert.Contains(t, dependencies["door_sensor_id"], "smart_siren_id")
	assert.Len(t, dependencies["door_sensor_id"], 3)

	assert.Contains(t, dependencies["camera2_id"], "smart_siren_id")
	assert.Len(t, dependencies["camera2_id"], 1)

	assert.Contains(t, dependencies["water_leak_sensor_id"], "smart_siren_id")
	assert.Len(t, dependencies["water_leak_sensor_id"], 1)
}
