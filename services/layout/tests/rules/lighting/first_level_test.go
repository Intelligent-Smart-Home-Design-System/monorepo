package lighting

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/stretchr/testify/assert"
)

func TestLightingLevel1(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "r1",
			Name: apartment.RoomLiving,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
		{
			ID:   "r2",
			Name: apartment.RoomKitchen,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
		{
			ID:   "r3",
			Name: apartment.RoomPassage,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 1, Y: 0},
				{X: 1, Y: 1},
				{X: 0, Y: 1},
			},
		},
		{
			ID:   "r4",
			Name: apartment.RoomBathroom,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 1, Y: 0},
				{X: 1, Y: 1},
				{X: 0, Y: 1},
			},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"lighting": "1",
	}

	st := storage.NewStorage()
	st.LoadAllLightingRules()

	err1 := configs.LoadTracksConfig("../../../internal/configs/tracks.json")
	err2 := configs.LoadDevicesConfig("../../../internal/configs/devices.json")
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	e := engine.NewEngine(st)
	layout, err := e.PlaceDevices(apartmentStruct, selectedLevels)
	assert.NoError(t, err)

	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r1"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r2"))
	assert.False(t, layout.HasDeviceInRoom("smart_bulb", "r3"))
	assert.False(t, layout.HasDeviceInRoom("smart_bulb", "r4"))
}

func TestLightingLevel1PriceCalculation(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "r1",
			Name: apartment.RoomLiving,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
		{
			ID:   "r2",
			Name: apartment.RoomKitchen,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"lighting": "1",
	}

	st := storage.NewStorage()
	st.LoadAllLightingRules()
	err1 := configs.LoadTracksConfig("../../../internal/configs/tracks.json")
	err2 := configs.LoadDevicesConfig("../../../internal/configs/devices.json")
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	e := engine.NewEngine(st)
	layout, err := e.PlaceDevices(apartmentStruct, selectedLevels)
	assert.NoError(t, err)

	priceInfo := e.CalculateLayoutPrice(layout)
	devicesConfig := configs.GetGlobalDevicesConfig()
	bulb := devicesConfig.Devices["smart_bulb"]

	assert.Equal(t, bulb.Price.Min*2, priceInfo.MinPrice)
	assert.Equal(t, bulb.Price.Max*2, priceInfo.MaxPrice)
}
