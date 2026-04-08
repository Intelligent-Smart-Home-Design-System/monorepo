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

func TestForthLevelSimpleScript(t *testing.T) {
	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "living",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 4, Y: 0},
				{X: 4, Y: 4},
				{X: 2, Y: 4},
				{X: 2, Y: 3},
				{X: 0, Y: 3},
			},
		},
		{
			ID:   "2",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
	}

	walls := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 4, Y: 0},
			},
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 4, Y: 0},
				{X: 4, Y: 4},
			},
		},
		{
			ID: "1",
			Points: []point.Point{
				{X: 4, Y: 4},
				{X: 2, Y: 4},
			},
		},
		{
			ID: "1",
			Points: []point.Point{
				{X: 2, Y: 4},
				{X: 2, Y: 3},
			},
		},
		{
			ID: "1",
			Points: []point.Point{
				{X: 2, Y: 3},
				{X: 0, Y: 3},
			},
		},
	}

	window := apartment.Window{
		ID: "1",
		Points: []point.Point{
			{X: 0, Y: 1.2},
			{X: 0, Y: 1.6},
		},
		Rooms: []string{"1"},
	}

	door := apartment.Door{
		ID: "1",
		Points: []point.Point{
			{X: 1, Y: 0},
			{X: 2, Y: 0},
		},
		Rooms: []string{"2"},
	}

	apartmentStruct := &apartment.Apartment{
		Walls:   walls,
		Windows: []apartment.Window{window},
		Doors:   []apartment.Door{door},
		Rooms:   rooms,
	}
	apartmentStruct.MakeDependency()

	selectedLevels := map[string]string{
		"security": "4",
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

	for roomID, roomPlacement := range globalPlacement.Placements {
		for _, devicePlacement := range roomPlacement {
			switch devicePlacement.Device.Type {
			case "camera":
				if roomID == "1" {
					assert.Equal(t, point.Point{X: 4, Y: 0}, devicePlacement.Place)
				} else {
					variants := []point.Point{
						{X: 0, Y: 3},
						{X: 3, Y: 3},
					}
					assert.Contains(t, variants, devicePlacement.Place)
				}
			}
		}
	}

	livingRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for key := range globalPlacement.Placements["1"] {
		livingRoomKeys = append(livingRoomKeys, key)
	}

	correctLivingRoomKeys := []string{"window_sensor", "motion_sensor", "camera"}
	for _, key := range correctLivingRoomKeys {
		assert.Contains(t, livingRoomKeys, key)
	}

	assert.Equal(t, len(correctLivingRoomKeys), len(livingRoomKeys))

	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["2"]))
	for key := range globalPlacement.Placements["2"] {
		hallRoomKeys = append(hallRoomKeys, key)
	}

	correctHallRoomKeys := []string{"smart_lock", "smart_doorbell", "door_sensor", "motion_sensor", "camera"}
	for _, key := range correctHallRoomKeys {
		assert.Contains(t, hallRoomKeys, key)
	}

	assert.Equal(t, len(correctHallRoomKeys), len(hallRoomKeys))
}

func TestForthLevelPriceCalculation(t *testing.T) {
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
		"security": "4",
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

	assert.Equal(t, 55500, priceInfo.MinPrice)
	assert.Equal(t, 83000, priceInfo.MaxPrice)
}
