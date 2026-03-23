package security

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/tests/rules"

	"github.com/stretchr/testify/assert"
)

func TestFifthLevelSimpleScript(t *testing.T) {
	rooms := []entities.Room{
		{
			ID:   "1",
			Name: "hall",
			Area: []entities.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
	}

	door := entities.Door{
		ID: "1",
		Points: []entities.Point{
			{X: 1, Y: 0},
			{X: 2, Y: 0},
		},
		Rooms: []string{"1"},
	}

	apartment := &entities.Apartment{
		Doors: []entities.Door{door},
		Rooms: rooms,
	}

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
	globalPlacement, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements["1"] {
		switch devicePlacement.Device.Type {
		case "security_alarm":
			assert.Equal(t, entities.Point{X: 1.5, Y: 1.5}, devicePlacement.Place)
		}
	}

	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for key := range globalPlacement.Placements["1"] {
		hallRoomKeys = append(hallRoomKeys, key)
	}

	correctHallRoomKeys := []string{
		"security_alarm",
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
	rooms := []entities.Room{
		{
			ID:   "1",
			Name: "bathroom",
			Area: []entities.Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
		{
			ID:   "2",
			Name: "kitchen",
			Area: []entities.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
		{
			ID:   "3",
			Name: "hall",
			Area: []entities.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
		{
			ID:   "4",
			Name: "living",
			Area: []entities.Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
	}

	windows := []entities.Window{
		{
			ID: "1",
			Points: []entities.Point{
				{X: 0, Y: 1},
				{X: 0, Y: 2},
			},
			Rooms: []string{"2"},
		},
		{
			ID: "2",
			Points: []entities.Point{
				{X: 0, Y: 1},
				{X: 0, Y: 2},
			},
			Rooms: []string{"4"},
		},
	}

	door := entities.Door{
		ID:     "1",
		Points: []entities.Point{{X: 1, Y: 0}, {X: 2, Y: 0}},
		Rooms:  []string{"3"},
	}

	apartment := &entities.Apartment{
		Windows: windows,
		Doors:   []entities.Door{door},
		Rooms:   rooms,
	}

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
	globalPlacement, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.NoError(t, err)

	priceInfo := engine.CalculateLayoutPrice(globalPlacement)

	assert.Equal(t, 58500, priceInfo.MinPrice)
	assert.Equal(t, 89000, priceInfo.MaxPrice)
}
