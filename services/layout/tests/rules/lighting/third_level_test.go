package lighting

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/stretchr/testify/assert"
)

func TestLightingLevel3(t *testing.T) {
	apartment := &apartment.Apartment{
		ID:     "a1",
		Tracks: []string{"lighting"},
		Rooms: []*apartment.Room{
			{ID: "r1", Name: apartment.RoomLiving},
			{ID: "r2", Name: apartment.RoomKitchen},
			{ID: "r3", Name: apartment.RoomPassage},
			{ID: "r4", Name: apartment.RoomBathroom},
			{ID: "r5", Name: apartment.RoomCabinet},
		},
	}
	selectedLevels := map[string]string{
		"lighting": "3",
	}

	st := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig("../../../internal/configs/tracks.json")
	devicesConfig, err2 := configs.LoadDevicesConfig("../../../internal/configs/devices.json")
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	e := engine.NewEngine(st, tracksConfig, devicesConfig)
	globalPlacement, err := e.PlaceDevices(apartment, selectedLevels)
	assert.NoError(t, err)

	assert.NotEmpty(t, globalPlacement.Placements)
	_, livingHasIllumination := globalPlacement.Placements["r1"]["illumination_sensor"]
	_, kitchenHasIllumination := globalPlacement.Placements["r2"]["illumination_sensor"]
	assert.True(t, livingHasIllumination)
	assert.True(t, kitchenHasIllumination)
}
