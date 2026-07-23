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

func TestLightingLevel3(t *testing.T) {
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
				{X: 6000, Y: 0},
				{X: 6000, Y: 1000},
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
		{
			ID:   "r5",
			Name: apartment.RoomCabinet,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2000, Y: 0},
				{X: 2000, Y: 2000},
				{X: 0, Y: 2000},
			},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"lighting": "3",
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

	assert.NotEmpty(t, layout.Placements)
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r1"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r2"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r3"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r4"))

	assert.True(t, layout.HasDeviceInRoom("illumination_sensor", "r1"))
	assert.True(t, layout.HasDeviceInRoom("illumination_sensor", "r2"))

	assert.True(t, layout.HasDeviceInRoom("motion_sensor", "r3"))
	assert.True(t, layout.HasDeviceInRoom("motion_sensor", "r4"))
}

func TestLightingLevel3MotionSensorCount(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "r3",
			Name: apartment.RoomPassage,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 6000, Y: 0},
				{X: 6000, Y: 1000},
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
	}

	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"lighting": "3",
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

	passageCount := 0
	for _, p := range layout.Placements["r3"] {
		if p.Device.Type == "motion_sensor" {
			passageCount++
		}
	}
	assert.Equal(t, 2, passageCount)

	bathroomCount := 0
	for _, p := range layout.Placements["r4"] {
		if p.Device.Type == "motion_sensor" {
			bathroomCount++
		}
	}
	assert.Equal(t, 1, bathroomCount)
}

func TestLightingLevel3PriceCalculation(t *testing.T) {
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
				{X: 6, Y: 0},
				{X: 6, Y: 1},
				{X: 0, Y: 1},
			},
		},
		{
			ID:   "r4",
			Name: apartment.RoomBathroom,
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 1.5, Y: 0},
				{X: 1.5, Y: 1.5},
				{X: 0, Y: 1.5},
			},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"lighting": "3",
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

	expectedMin :=
		devicesConfig.Devices["smart_bulb"].Price.Min*4 +
			devicesConfig.Devices["motion_sensor"].Price.Min*3 +
			devicesConfig.Devices["illumination_sensor"].Price.Min*2

	expectedMax :=
		devicesConfig.Devices["smart_bulb"].Price.Max*4 +
			devicesConfig.Devices["motion_sensor"].Price.Max*3 +
			devicesConfig.Devices["illumination_sensor"].Price.Max*2

	assert.Equal(t, expectedMin, priceInfo.MinPrice)
	assert.Equal(t, expectedMax, priceInfo.MaxPrice)
}
