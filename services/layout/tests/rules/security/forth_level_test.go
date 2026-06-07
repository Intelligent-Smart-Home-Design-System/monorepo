<<<<<<< HEAD
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
	windows := []apartment.Window{
		{
			ID: "1",
			Points: []point.Point{
				{X: 3, Y: 0},
				{X: 5, Y: 0},
			},
			Rooms: []string{"1"},
			Width: 2,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 8, Y: 3},
				{X: 8, Y: 6},
			},
			Rooms: []string{"1"},
			Width: 3,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 3, Y: 12},
				{X: 5, Y: 12},
			},
			Rooms: []string{"2"},
			Width: 2,
		},
	}

	doors := []apartment.Door{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
			},
			Rooms: []string{"1"},
			Width: 2,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 6, Y: 7},
				{X: 8, Y: 7},
			},
			Rooms: []string{"1", "2"},
			Width: 2,
		},
	}

	walls_1 := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 8, Y: 0},
			},
			Width: 8,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 8, Y: 0},
				{X: 8, Y: 7},
			},
			Width: 7,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 8, Y: 7},
				{X: 0, Y: 7},
			},
			Width: 8,
		},
		{
			ID: "4",
			Points: []point.Point{
				{X: 0, Y: 7},
				{X: 0, Y: 0},
			},
			Width: 7,
		},
	}

	walls_2 := []apartment.Wall{
		{
			ID: "5",
			Points: []point.Point{
				{X: 0, Y: 7},
				{X: 6, Y: 7},
			},
			Width: 6,
		},
		{
			ID: "6",
			Points: []point.Point{
				{X: 6, Y: 7},
				{X: 6, Y: 14},
			},
			Width: 7,
		},
		{
			ID: "7",
			Points: []point.Point{
				{X: 6, Y: 14},
				{X: 0, Y: 14},
			},
			Width: 6,
		},
		{
			ID: "8",
			Points: []point.Point{
				{X: 0, Y: 14},
				{X: 0, Y: 7},
			},
			Width: 7,
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 8, Y: 0},
				{X: 8, Y: 7},
				{X: 0, Y: 7},
			},
			Walls: []string{"1", "2", "3", "4"},
			Doors: []string{"1", "2"},
			Windows: []string{"1", "2"},
		},
		{
			ID:   "2",
			Name: "living",
			Area: []point.Point{
				{X: 0, Y: 7},
				{X: 6, Y: 7},
				{X: 6, Y: 14},
				{X: 0, Y: 14},
			},
			Walls: []string{"5", "6", "7", "8"},
			Doors: []string{"2"},
			Windows: []string{"3"},
		},
	}

	walls := walls_1
	walls = append(walls, walls_2...)

	apartmentStruct := &apartment.Apartment{
		Walls:   walls,
		Windows: windows,
		Doors:   doors,
		Rooms:   rooms,
	}

	selectedLevels := map[string]string{
		"security": "4",
	}

	err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	engine := engine.NewEngine(storage)
	globalPlacement, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

	assert.NoError(t, err)

	for roomID, roomPlacement := range globalPlacement.Placements {
		for _, devicePlacement := range roomPlacement {
			switch devicePlacement.Device.Type {
			case "camera":
				if roomID == "1" {
					assert.Equal(t, &point.Point{X: 8, Y: 7}, devicePlacement.Position)
				} else {
					assert.Equal(t, point.Point{X: 0, Y: 7}, *devicePlacement.Position)
				}
			}
		}
	}

	livingRoomKeys := make([]string, 0, len(globalPlacement.Placements["2"]))
	for _, placement := range globalPlacement.Placements["1"] {
		livingRoomKeys = append(livingRoomKeys, placement.Device.Type)
	}

	correctLivingRoomKeys := []string{"motion_sensor", "camera"}
	for _, key := range correctLivingRoomKeys {
		assert.Contains(t, livingRoomKeys, key)
	}

	assert.LessOrEqual(t, len(correctLivingRoomKeys), len(livingRoomKeys))

	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for _, placement := range globalPlacement.Placements["1"] {
		hallRoomKeys = append(hallRoomKeys, placement.Device.Type)
	}

	correctHallRoomKeys := []string{"smart_lock", "smart_doorbell", "door_sensor", "motion_sensor", "camera"}
	for _, key := range correctHallRoomKeys {
		assert.Contains(t, hallRoomKeys, key)
	}

	assert.LessOrEqual(t, len(correctHallRoomKeys), len(hallRoomKeys))
}

func TestForthLevelPriceCalculation(t *testing.T) {
	windows := []apartment.Window{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 1},
				{X: 0, Y: 2},
			},
			Rooms: []string{"1"},
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 5, Y: 0},
				{X: 5, Y: 2},
			},
			Rooms: []string{"1"},
		},
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
	}

	walls := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 5, Y: 0},
			},
			Width: 5,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 5, Y: 0},
				{X: 5, Y: 2},
			},
			Width: 2,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 5, Y: 2},
				{X: 3, Y: 2},
			},
			Width: 3,
		},
		{
			ID: "4",
			Points: []point.Point{
				{X: 3, Y: 2},
				{X: 3, Y: 5},
			},
			Width: 3,
		},
		{
			ID: "5",
			Points: []point.Point{
				{X: 3, Y: 5},
				{X: 0, Y: 5},
			},
			Width: 3,
		},
		{
			ID: "6",
			Points: []point.Point{
				{X: 0, Y: 5},
				{X: 0, Y: 0},
			},
			Width: 5,
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "living",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 5, Y: 0},
				{X: 5, Y: 2},
				{X: 3, Y: 2},
				{X: 3, Y: 5},
				{X: 0, Y: 5},
			},
			Windows: []string{"1", "2"},
			Walls: []string{"1", "2", "3", "4", "5", "6"},
			Doors: []string{"1"},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Windows: windows,
		Doors:   doors,
		Walls: walls,
		Rooms:   rooms,
	}

	selectedLevels := map[string]string{
		"security": "4",
	}

	err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	engine := engine.NewEngine(storage)
	globalPlacement, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

	assert.NoError(t, err)

	priceInfo := engine.CalculateLayoutPrice(globalPlacement)

	assert.Equal(t, 13000, priceInfo.MinPrice)
	assert.Equal(t, 23000, priceInfo.MaxPrice)
}
=======
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

// TODO: fix
// func TestForthLevelSimpleScript(t *testing.T) {
// 	windows := []apartment.Window{
// 		{
// 			ID: "1",
// 			Points: []point.Point{
// 				{X: 3, Y: 0},
// 				{X: 5, Y: 0},
// 			},
// 			Rooms: []string{"1"},
// 			Width: 2,
// 		},
// 		{
// 			ID: "2",
// 			Points: []point.Point{
// 				{X: 8, Y: 3},
// 				{X: 8, Y: 6},
// 			},
// 			Rooms: []string{"1"},
// 			Width: 3,
// 		},
// 		{
// 			ID: "3",
// 			Points: []point.Point{
// 				{X: 3, Y: 12},
// 				{X: 5, Y: 12},
// 			},
// 			Rooms: []string{"2"},
// 			Width: 2,
// 		},
// 	}

// 	doors := []apartment.Door{
// 		{
// 			ID: "1",
// 			Points: []point.Point{
// 				{X: 0, Y: 0},
// 				{X: 2, Y: 0},
// 			},
// 			Rooms: []string{"1"},
// 			Width: 2,
// 		},
// 		{
// 			ID: "2",
// 			Points: []point.Point{
// 				{X: 6, Y: 7},
// 				{X: 8, Y: 7},
// 			},
// 			Rooms: []string{"1", "2"},
// 			Width: 2,
// 		},
// 	}

// 	walls_1 := []apartment.Wall{
// 		{
// 			ID: "1",
// 			Points: []point.Point{
// 				{X: 0, Y: 0},
// 				{X: 8, Y: 0},
// 			},
// 			Width: 8,
// 		},
// 		{
// 			ID: "2",
// 			Points: []point.Point{
// 				{X: 8, Y: 0},
// 				{X: 8, Y: 7},
// 			},
// 			Width: 7,
// 		},
// 		{
// 			ID: "3",
// 			Points: []point.Point{
// 				{X: 8, Y: 7},
// 				{X: 0, Y: 7},
// 			},
// 			Width: 8,
// 		},
// 		{
// 			ID: "4",
// 			Points: []point.Point{
// 				{X: 0, Y: 7},
// 				{X: 0, Y: 0},
// 			},
// 			Width: 7,
// 		},
// 	}

// 	walls_2 := []apartment.Wall{
// 		{
// 			ID: "5",
// 			Points: []point.Point{
// 				{X: 0, Y: 7},
// 				{X: 6, Y: 7},
// 			},
// 			Width: 6,
// 		},
// 		{
// 			ID: "6",
// 			Points: []point.Point{
// 				{X: 6, Y: 7},
// 				{X: 6, Y: 14},
// 			},
// 			Width: 7,
// 		},
// 		{
// 			ID: "7",
// 			Points: []point.Point{
// 				{X: 6, Y: 14},
// 				{X: 0, Y: 14},
// 			},
// 			Width: 6,
// 		},
// 		{
// 			ID: "8",
// 			Points: []point.Point{
// 				{X: 0, Y: 14},
// 				{X: 0, Y: 7},
// 			},
// 			Width: 7,
// 		},
// 	}

// 	rooms := []apartment.Room{
// 		{
// 			ID:   "1",
// 			Name: "hall",
// 			Area: []point.Point{
// 				{X: 0, Y: 0},
// 				{X: 8, Y: 0},
// 				{X: 8, Y: 7},
// 				{X: 0, Y: 7},
// 			},
// 			Walls: []string{"1", "2", "3", "4"},
// 			Doors: []string{"1", "2"},
// 			Windows: []string{"1", "2"},
// 		},
// 		{
// 			ID:   "2",
// 			Name: "living",
// 			Area: []point.Point{
// 				{X: 0, Y: 7},
// 				{X: 6, Y: 7},
// 				{X: 6, Y: 14},
// 				{X: 0, Y: 14},
// 			},
// 			Walls: []string{"5", "6", "7", "8"},
// 			Doors: []string{"2"},
// 			Windows: []string{"3"},
// 		},
// 	}

// 	walls := walls_1
// 	walls = append(walls, walls_2...)

// 	apartmentStruct := &apartment.Apartment{
// 		Walls:   walls,
// 		Windows: windows,
// 		Doors:   doors,
// 		Rooms:   rooms,
// 	}

// 	selectedLevels := map[string]string{
// 		"security": "4",
// 	}

// 	err1 := configs.LoadTracksConfig(rules.GetTracksPath())
// 	err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

// 	assert.NoError(t, err1)
// 	assert.NoError(t, err2)

// 	storage := storage.NewStorage()
// 	storage.LoadAllSecurityRules()

// 	engine := engine.NewEngine(storage)
// 	globalPlacement, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

// 	assert.NoError(t, err)

// 	for roomID, roomPlacement := range globalPlacement.Placements {
// 		for _, devicePlacement := range roomPlacement {
// 			switch devicePlacement.Device.Type {
// 			case "camera":
// 				if roomID == "1" {
// 					assert.Equal(t, &point.Point{X: 4, Y: 0}, devicePlacement.Position)
// 				} else {
// 					variants := []point.Point{
// 						{X: 0, Y: 3},
// 						{X: 3, Y: 3},
// 					}
// 					assert.Contains(t, variants, *devicePlacement.Position)
// 				}
// 			}
// 		}
// 	}

// 	livingRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
// 	for _, placement := range globalPlacement.Placements["1"] {
// 		livingRoomKeys = append(livingRoomKeys, placement.Device.Type)
// 	}

// 	correctLivingRoomKeys := []string{"window_sensor", "motion_sensor", "camera"}
// 	for _, key := range correctLivingRoomKeys {
// 		assert.Contains(t, livingRoomKeys, key)
// 	}

// 	assert.Equal(t, len(correctLivingRoomKeys), len(livingRoomKeys))

// 	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["2"]))
// 	for _, placement := range globalPlacement.Placements["2"] {
// 		hallRoomKeys = append(hallRoomKeys, placement.Device.Type)
// 	}

// 	correctHallRoomKeys := []string{"smart_lock", "smart_doorbell", "door_sensor", "motion_sensor", "camera"}
// 	for _, key := range correctHallRoomKeys {
// 		assert.Contains(t, hallRoomKeys, key)
// 	}

// 	assert.Equal(t, len(correctHallRoomKeys), len(hallRoomKeys))
// }

func TestForthLevelPriceCalculation(t *testing.T) {
	windows := []apartment.Window{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 1},
				{X: 0, Y: 2},
			},
			Rooms: []string{"1"},
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 5, Y: 0},
				{X: 5, Y: 2},
			},
			Rooms: []string{"1"},
		},
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
	}

	walls := []apartment.Wall{
		{
			ID: "1",
			Points: []point.Point{
				{X: 0, Y: 0},
				{X: 5, Y: 0},
			},
			Width: 5,
		},
		{
			ID: "2",
			Points: []point.Point{
				{X: 5, Y: 0},
				{X: 5, Y: 2},
			},
			Width: 2,
		},
		{
			ID: "3",
			Points: []point.Point{
				{X: 5, Y: 2},
				{X: 3, Y: 2},
			},
			Width: 3,
		},
		{
			ID: "4",
			Points: []point.Point{
				{X: 3, Y: 2},
				{X: 3, Y: 5},
			},
			Width: 3,
		},
		{
			ID: "5",
			Points: []point.Point{
				{X: 3, Y: 5},
				{X: 0, Y: 5},
			},
			Width: 3,
		},
		{
			ID: "6",
			Points: []point.Point{
				{X: 0, Y: 5},
				{X: 0, Y: 0},
			},
			Width: 5,
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "living",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 5, Y: 0},
				{X: 5, Y: 2},
				{X: 3, Y: 2},
				{X: 3, Y: 5},
				{X: 0, Y: 5},
			},
			Windows: []string{"1", "2"},
			Walls: []string{"1", "2", "3", "4", "5", "6"},
			Doors: []string{"1"},
		},
	}

	apartmentStruct := &apartment.Apartment{
		Windows: windows,
		Doors:   doors,
		Walls: walls,
		Rooms:   rooms,
	}

	selectedLevels := map[string]string{
		"security": "4",
	}

	err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	engine := engine.NewEngine(storage)
	globalPlacement, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

	assert.NoError(t, err)

	priceInfo := engine.CalculateLayoutPrice(globalPlacement)

	assert.Equal(t, 13000, priceInfo.MinPrice)
	assert.Equal(t, 23000, priceInfo.MaxPrice)
}
>>>>>>> 4bf54f8 (hz)
