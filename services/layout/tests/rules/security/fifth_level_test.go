package security

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/tests/rules"

	"github.com/stretchr/testify/assert"
)

func TestFifthLevelSimpleScript(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
	}

	door := apartment.Door{
		ID: "1",
		Points: []point.Point{
			{X: 1, Y: 0},
			{X: 2, Y: 0},
		},
		Rooms: []string{"1"},
	}

	apartmentStruct := &apartment.Apartment{
		Doors: []apartment.Door{door},
		Rooms: rooms,
	}
	apartmentStruct.MakeDependency()

	selectedLevels := map[string]string{
		"security": "5",
	}

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	tracksConfig, err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	globalPlacement, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements["1"] {
		switch devicePlacement.Device.Type {
		case "smart_siren":
			assert.Equal(t, &point.Point{X: 1.5, Y: 1.5}, devicePlacement.Place)
		}
	}

	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for key := range globalPlacement.Placements["1"] {
		hallRoomKeys = append(hallRoomKeys, key)
	}

	correctHallRoomKeys := []string{
		"smart_siren",
		"motion_sensor",
		"door_sensor",
		"camera",
		"smart_doorbell",
		"smart_lock",
	}
	for _, key := range correctHallRoomKeys {
		assert.Contains(t, hallRoomKeys, key)
	}

	assert.Equal(t, len(correctHallRoomKeys), len(hallRoomKeys))
}

func TestFifthLevelPriceCalculation(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "bathroom",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
		{
			ID:   "2",
			Name: "kitchen",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
		{
			ID:   "3",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
		{
			ID:   "4",
			Name: "living",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
	}

	windows := []apartment.Window{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 1},
				{X: 0, Y: 2},
			},
			Rooms: []string{"2"},
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 0, Y: 1},
				{X: 0, Y: 2},
			},
			Rooms: []string{"4"},
		},
	}

	door := apartment.Door{
		ID:     "1",
		Points: []point.Point{{X: 1, Y: 0}, {X: 2, Y: 0}},
		Rooms:  []string{"3"},
	}

	apartmentStruct := &apartment.Apartment{
		Windows: windows,
		Doors:   []apartment.Door{door},
		Rooms:   rooms,
	}
	apartmentStruct.MakeDependency()

	selectedLevels := map[string]string{
		"security": "5",
	}

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	tracksConfig, err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	globalPlacement, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

	assert.NoError(t, err)

	priceInfo := engine.CalculateLayoutPrice(globalPlacement)

	assert.Equal(t, 58500, priceInfo.MinPrice)
	assert.Equal(t, 89000, priceInfo.MaxPrice)
}
