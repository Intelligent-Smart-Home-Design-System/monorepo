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

func TestSimpleFirstLevelScript(t *testing.T) {
	room := apartment.Room{
		ID:   "1",
		Name: "kitchen",
		Area: []point.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}
	apartmentStruct := &apartment.Apartment{
		Rooms: []apartment.Room{room},
	}
	apartmentStruct.MakeDependency()

	selectedLevels := map[string]string{
		"security": "1",
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

	for _, devicePlacement := range globalPlacement.Placements[room.ID] {
		switch devicePlacement.Device.Type {
		case "water_leak_sensor":
			assert.Equal(t, point.Point{X: 1, Y: 1}, devicePlacement.Place)
		case "gas_leak_sensor":
			assert.Equal(t, point.Point{X: 1, Y: 1}, devicePlacement.Place)
		}
	}

	kitchenRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for key := range globalPlacement.Placements["1"] {
		kitchenRoomKeys = append(kitchenRoomKeys, key)
	}

	correctKitchenRoomKeys := []string{"water_leak_sensor", "gas_leak_sensor"}
	for _, key := range correctKitchenRoomKeys {
		assert.Contains(t, kitchenRoomKeys, key)
	}

	assert.Equal(t, len(correctKitchenRoomKeys), len(kitchenRoomKeys))
}

func TestMultipleRooms(t *testing.T) {
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
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
	}
	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}
	apartmentStruct.MakeDependency()

	selectedLevels := map[string]string{
		"security": "1",
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

	for _, devicePlacement := range globalPlacement.Placements[rooms[0].ID] {
		switch devicePlacement.Device.Type {
		case "water_leak_sensor":
			assert.Equal(t, point.Point{X: 1, Y: 1}, devicePlacement.Place)
		case "gas_leak_sensor":
			assert.Equal(t, point.Point{X: 1, Y: 1}, devicePlacement.Place)
		}
	}

	for _, devicePlacement := range globalPlacement.Placements[rooms[1].ID] {
		switch devicePlacement.Device.Type {
		case "water_leak_sensor":
			assert.Equal(t, point.Point{X: 1.5, Y: 1.5}, devicePlacement.Place)
		}
	}

	assert.Equal(t, 0, len(globalPlacement.Placements[rooms[2].ID]))
}

func TestFirstLevelPriceCalculation(t *testing.T) {
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
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
		},
	}
	apartmentStruct := &apartment.Apartment{
		Rooms: rooms,
	}
	apartmentStruct.MakeDependency()

	selectedLevels := map[string]string{
		"security": "1",
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

	assert.Equal(t, 4000, priceInfo.MinPrice)
	assert.Equal(t, 9000, priceInfo.MaxPrice)
}
