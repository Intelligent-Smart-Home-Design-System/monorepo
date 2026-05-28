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

func TestThirdLevelSimpleScript(t *testing.T) {
	window := apartment.Window{
		ID: "1",
		Points: []point.Point{
			{X: 0, Y: 1.2},
			{X: 0, Y: 1.6},
		},
		Rooms: []string{"1"},
		Width: 0.4,
	}

	doors := []apartment.Door{
		{
			ID: "1",
			Points: []point.Point{
				{X: 1, Y: 0},
				{X: 3, Y: 0},
			},
			Rooms: []string{"1"},
			Width: 2,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 12, Y: 3},
				{X: 12, Y: 4},
			},
			Rooms: []string{"1", "2"},
			Width: 1,
		},
	}

	walls_1 := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 12, Y: 0},
			},
			Width: 12,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 12, Y: 0},
				{X: 12, Y: 5},
			},
			Width: 5,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 12, Y: 5},
				{X: 0, Y: 5},
			},
			Width: 12,
		},
		{
			ID: "4",
			Points: []point.Point{
				{X: 0, Y: 5},
				{X: 0, Y: 0},
			},
			Width: 5,
		},
	}

	walls_2 := []apartment.Wall{
		{
			ID: "5",
			Points: []point.Point{
				{X: 12, Y: 0},
				{X: 16, Y: 0},
			},
			Width: 4,
		},
		{
			ID: "6",
			Points: []point.Point{
				{X: 16, Y: 0},
				{X: 16, Y: 20},
			},
			Width: 20,
		},
		{
			ID: "7",
			Points: []point.Point{
				{X: 16, Y: 20},
				{X: 12, Y: 20},
			},
			Width: 4,
		},
		{
			ID: "8",
			Points: []point.Point{
				{X: 12, Y: 20},
				{X: 12, Y: 0},
			},
			Width: 20,
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "living",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 12, Y: 0},
				{X: 12, Y: 5},
				{X: 0, Y: 5},
			},
			Doors:   []string{doors[1].ID},
			Windows: []string{window.ID},
			Walls:   []string{"1", "2", "3", "4"},
		},
		{
			ID:   "2",
			Name: "hall",
			Area: []point.Point{
				{X: 12, Y: 0},
				{X: 16, Y: 0},
				{X: 16, Y: 20},
				{X: 12, Y: 20},
			},
			Doors: []string{doors[0].ID},
			Walls: []string{"5", "6", "7", "8"},
		},
	}

	walls := walls_1
	walls = append(walls, walls_2...)

	ap := &apartment.Apartment{
		Windows: []apartment.Window{window},
		Doors:   doors,
		Rooms:   rooms,
		Walls:   walls,
	}

	selectedLevels := map[string]string{
		"security": "3",
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

	for roomID, roomPlacement := range globalPlacement.Placements {
		for _, devicePlacement := range roomPlacement {
			switch devicePlacement.Device.Type {
			case "door_sensor":
				if roomID == "1" {
					assert.Equal(t, &point.Point{X: 2, Y: 0}, devicePlacement.Position)
				} else {
					assert.Equal(t, &point.Point{X: 2, Y: 0}, devicePlacement.Position)
				}
			case "window_sensor":
				assert.Equal(t, &point.Point{X: 0, Y: 1.4}, devicePlacement.Position)
			case "motion_sensor":
				assert.Equal(t, &point.Point{X: 0, Y: 0}, devicePlacement.Position)
			}
		}
	}

	livingRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for _, placement := range globalPlacement.Placements["1"] {
		livingRoomKeys = append(livingRoomKeys, placement.Device.Type)
	}

	correctLivingRoomKeys := []string{"window_sensor", "motion_sensor"}
	for _, key := range correctLivingRoomKeys {
		assert.Contains(t, livingRoomKeys, key)
	}

	assert.Equal(t, len(correctLivingRoomKeys), len(livingRoomKeys))

	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["2"]))
	for _, placement := range globalPlacement.Placements["2"] {
		hallRoomKeys = append(hallRoomKeys, placement.Device.Type)
	}

	correctHallRoomKeys := []string{"smart_lock", "smart_doorbell", "door_sensor"}
	for _, key := range correctHallRoomKeys {
		assert.Contains(t, hallRoomKeys, key)
	}

	assert.Equal(t, len(correctHallRoomKeys), len(hallRoomKeys))
}

func TestThirdLevelPriceCalculation(t *testing.T) {
	window := apartment.Window{
		ID: "1",
		Points: []point.Point{
			{X: 0, Y: 1.2},
			{X: 0, Y: 1.6},
		},
		Rooms: []string{"1"},
	}

	doors := []apartment.Door{
		{
			ID: "1",
			Points: []point.Point{
				{X: 1, Y: 0},
				{X: 3, Y: 0},
			},
			Rooms: []string{"1"},
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 12, Y: 3},
				{X: 12, Y: 4},
			},
			Rooms: []string{"1", "2"},
		},
	}

	walls_1 := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 12, Y: 0},
			},
			Width: 2,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 12, Y: 0},
				{X: 12, Y: 5},
			},
			Width: 2,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 12, Y: 5},
				{X: 0, Y: 5},
			},
			Width: 2,
		},
		{
			ID: "4",
			Points: []point.Point{
				{X: 0, Y: 5},
				{X: 0, Y: 0},
			},
			Width: 2,
		},
	}

	walls_2 := []apartment.Wall{
		{
			ID: "5",
			Points: []point.Point{
				{X: 12, Y: 0},
				{X: 16, Y: 0},
			},
			Width: 2,
		},
		{
			ID: "6",
			Points: []point.Point{
				{X: 16, Y: 0},
				{X: 16, Y: 20},
			},
			Width: 2,
		},
		{
			ID: "7",
			Points: []point.Point{
				{X: 16, Y: 20},
				{X: 12, Y: 20},
			},
			Width: 2,
		},
		{
			ID: "8",
			Points: []point.Point{
				{X: 12, Y: 20},
				{X: 12, Y: 0},
			},
			Width: 2,
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "living",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 12, Y: 0},
				{X: 12, Y: 5},
				{X: 0, Y: 5},
			},
			Doors:   []string{doors[1].ID},
			Windows: []string{window.ID},
			Walls:   []string{"1", "2", "3", "4"},
		},
		{
			ID:   "2",
			Name: "hall",
			Area: []point.Point{
				{X: 12, Y: 0},
				{X: 16, Y: 0},
				{X: 16, Y: 20},
				{X: 12, Y: 20},
			},
			Doors: []string{doors[0].ID},
			Walls: []string{"5", "6", "7", "8"},
		},
	}

	walls := walls_1
	walls = append(walls, walls_2...)

	ap := &apartment.Apartment{
		Windows: []apartment.Window{window},
		Doors:   doors,
		Rooms:   rooms,
		Walls:   walls,
	}

	selectedLevels := map[string]string{
		"security": "3",
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

	assert.Equal(t, 32000, priceInfo.MinPrice)
	assert.Equal(t, 41000, priceInfo.MaxPrice)
}
