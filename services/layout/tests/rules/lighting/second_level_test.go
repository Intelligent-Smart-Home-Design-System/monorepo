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

func TestLightingLevel2(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "r1",
			Name: apartment.RoomLiving,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2000, Y: 0},
				{X: 2000, Y: 2000},
				{X: 0, Y: 2000},
			},
		},
		{
			ID:   "r2",
			Name: apartment.RoomKitchen,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
		},
		{
			ID:   "r3",
			Name: apartment.RoomPassage,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 5000, Y: 0},
				{X: 5000, Y: 1000},
				{X: 0, Y: 1000},
			},
		},
		{
			ID:   "r4",
			Name: apartment.RoomBathroom,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 1500, Y: 0},
				{X: 1500, Y: 1500},
				{X: 0, Y: 1500},
			},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"lighting": "2",
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
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r3"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r4"))
}

func TestLightingLevel2PriceCalculation(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "r1",
			Name: apartment.RoomLiving,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2000, Y: 0},
				{X: 2000, Y: 2000},
				{X: 0, Y: 2000},
			},
		},
		{
			ID:   "r2",
			Name: apartment.RoomKitchen,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
		},
		{
			ID:   "r3",
			Name: apartment.RoomPassage,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 5000, Y: 0},
				{X: 5000, Y: 1000},
				{X: 0, Y: 1000},
			},
		},
		{
			ID:   "r4",
			Name: apartment.RoomBathroom,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 1500, Y: 0},
				{X: 1500, Y: 1500},
				{X: 0, Y: 1500},
			},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"lighting": "2",
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

	assert.Equal(t, bulb.Price.Min*4, priceInfo.MinPrice)
	assert.Equal(t, bulb.Price.Max*4, priceInfo.MaxPrice)
}
