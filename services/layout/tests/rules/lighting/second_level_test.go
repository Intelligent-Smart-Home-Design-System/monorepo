package lighting

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/stretchr/testify/assert"
)

func TestLightingLevel2(t *testing.T) {
	apartment := &entities.Apartment{
		ID:     "a1",
		Tracks: []string{"lighting"},
		Rooms: []*entities.Room{
			{ID: "r1", Name: "living"},
			{ID: "r2", Name: "kitchen"},
			{ID: "r3", Name: "passage"},
			{ID: "r4", Name: "bathroom"},
			{ID: "r5", Name: "cabinet"},
		},
	}
	selectedLevels := map[string]string{
		"lighting": "2",
	}

	st := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig("../../../internal/configs/tracks.json")
	devicesConfig, err2 := configs.LoadDevicesConfig("../../../internal/configs/devices.json")
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	e := engine.NewEngine(st, tracksConfig, devicesConfig)
	globalPlacement, err := e.PlaceDevices(apartment, selectedLevels)
	assert.NoError(t, err)

	_, passageHasBulb := globalPlacement.Placements["r3"]["smart_bulb"]
	_, bathroomHasBulb := globalPlacement.Placements["r4"]["smart_bulb"]
	assert.True(t, passageHasBulb)
	assert.True(t, bathroomHasBulb)
}
