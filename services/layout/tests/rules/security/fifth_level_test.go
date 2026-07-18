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
	doors := []apartment.Door{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 1000, Y: 0},
			},
			Rooms: []string{"1"},
		},
	}

	walls := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
			},
			Width: 3000,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
			},
			Width: 3000,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
			Width: 3000,
		},
		{
			ID: "4",
			Points: []point.Point{
				{X: 0, Y: 3000},
				{X: 0, Y: 0},
			},
			Width: 3000,
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
			Doors: []string{"1"},
			Walls: []string{"1", "2", "3", "4"},
		},
	}

	ap := &apartment.Apartment{
		Doors: doors,
		Walls: walls,
		Rooms: rooms,
	}

	selectedLevels := map[string]string{
		"security": "5",
	}

	err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	engine := engine.NewEngine(storage)
	globalPlacement, err := engine.PlaceDevices(ap, selectedLevels)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements["1"] {
		switch devicePlacement.Device.Type {
		case "smart_siren":
			assert.Equal(t, &point.Point{X: 1500, Y: 1500}, devicePlacement.Position)
		}
	}

	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for _, placement := range globalPlacement.Placements["1"] {
		hallRoomKeys = append(hallRoomKeys, placement.Device.Type)
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
	doors := []apartment.Door{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 1000, Y: 0},
			},
			Rooms: []string{"1"},
		},
	}

	walls := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
			},
			Width: 3000,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
			},
			Width: 3000,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
			Width: 3000,
		},
		{
			ID: "4",
			Points: []point.Point{
				{X: 0, Y: 3000},
				{X: 0, Y: 0},
			},
			Width: 3000,
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
			Doors: []string{"1"},
			Walls: []string{"1", "2", "3", "4"},
		},
	}

	ap := &apartment.Apartment{
		Doors:   doors,
		Walls: walls,
		Rooms:   rooms,
	}

	selectedLevels := map[string]string{
		"security": "5",
	}

	err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	engine := engine.NewEngine(storage)
	globalPlacement, err := engine.PlaceDevices(ap, selectedLevels)

	assert.NoError(t, err)

	priceInfo := engine.CalculateLayoutPrice(globalPlacement)

	assert.Equal(t, 41500, priceInfo.MinPrice)
	assert.Equal(t, 57000, priceInfo.MaxPrice)
}
