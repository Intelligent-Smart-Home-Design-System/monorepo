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
	stove := apartment.Furniture{
		ID:   "1",
		Category: apartment.Stove,
		Points: []point.Point{
			{X: 2000, Y: 2000},
			{X: 2000, Y: 3000},
			{X: 3000, Y: 3000},
			{X: 3000, Y: 2000},
		},
	}

	sink := apartment.Furniture{
		ID:   "2",
		Category: apartment.Sink,
		Points: []point.Point{
			{X: 1000, Y: 2000},
			{X: 1000, Y: 3000},
			{X: 2000, Y: 3000},
			{X: 2000, Y: 2000},
		},
	}

	room := apartment.Room{
		ID:   "1",
		Name: apartment.RoomKitchen,
		Area: []point.Point{
			{X: 0, Y: 0},
			{X: 3000, Y: 0},
			{X: 3000, Y: 3000},
			{X: 0, Y: 3000},
		},
		Furniture: []string{"1", "2"},
	}
	ap := &apartment.Apartment{
		Rooms:      []apartment.Room{room},
		Furniture:   []apartment.Furniture{sink, stove},
	}

	selectedLevels := map[string]string{
		"security": "1",
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

	for _, devicePlacement := range globalPlacement.Placements[room.ID] {
		switch devicePlacement.Device.Type {
		case "water_leak_sensor":
			assert.Equal(t, &point.Point{X: 1500, Y: 2500}, devicePlacement.Position)
		case "gas_leak_sensor":
			assert.Equal(t, &point.Point{X: 2500, Y: 2500}, devicePlacement.Position)
		}
	}

	kitchenRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for _, placement := range globalPlacement.Placements["1"] {
		kitchenRoomKeys = append(kitchenRoomKeys, placement.Device.Type)
	}

	correctKitchenRoomKeys := []string{"water_leak_sensor", "gas_leak_sensor"}
	for _, key := range correctKitchenRoomKeys {
		assert.Contains(t, kitchenRoomKeys, key)
	}

	assert.Equal(t, len(correctKitchenRoomKeys), len(kitchenRoomKeys))
}

func TestMultipleRooms(t *testing.T) {
	stove := apartment.Furniture{
		ID:   "1",
		Category: apartment.Stove,
		Points: []point.Point{
			{X: 0, Y: 0},
			{X: 0, Y: 1000},
			{X: 1000, Y: 1000},
			{X: 1000, Y: 0},
		},
	}

	sink := apartment.Furniture{
		ID:   "2",
		Category: apartment.Sink,
		Points: []point.Point{
			{X: 1000, Y: 2000},
			{X: 1000, Y: 3000},
			{X: 2000, Y: 3000},
			{X: 2000, Y: 2000},
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "bathroom",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2000, Y: 0},
				{X: 2000, Y: 2000},
				{X: 0, Y: 2000},
			},
			Furniture: []string{"2"},

		},
		{
			ID:   "2",
			Name: "kitchen",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
			Furniture: []string{"1"},
		},
		{
			ID:   "3",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2000, Y: 0},
				{X: 2000, Y: 2000},
				{X: 0, Y: 2000},
			},
		},
	}

	ap := &apartment.Apartment{
		Rooms: rooms,
		Furniture: []apartment.Furniture{sink, stove},
	}

	selectedLevels := map[string]string{
		"security": "1",
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

	for _, devicePlacement := range globalPlacement.Placements[rooms[0].ID] {
		switch devicePlacement.Device.Type {
		case "water_leak_sensor":
			assert.Equal(t, &point.Point{X: 1500, Y: 2500}, devicePlacement.Position)
	}

	for _, devicePlacement := range globalPlacement.Placements[rooms[1].ID] {
		switch devicePlacement.Device.Type {
		case "water_leak_sensor":
			assert.Equal(t, &point.Point{X: 1500, Y: 2500}, devicePlacement.Position)
		case "gas_leak_sensor":
			assert.Equal(t, &point.Point{X: 500, Y: 500}, devicePlacement.Position)
		}
		}
	}

	assert.Equal(t, 0, len(globalPlacement.Placements[rooms[2].ID]))
}

func TestFirstLevelPriceCalculation(t *testing.T) {
	stove := apartment.Furniture{
		ID:   "1",
		Category: apartment.Stove,
		Points: []point.Point{
			{X: 0, Y: 0},
			{X: 0, Y: 1000},
			{X: 1000, Y: 1000},
			{X: 1000, Y: 0},
		},
	}

	sink := apartment.Furniture{
		ID:   "2",
		Category: apartment.Sink,
		Points: []point.Point{
			{X: 1000, Y: 2000},
			{X: 1000, Y: 3000},
			{X: 2000, Y: 3000},
			{X: 2000, Y: 2000},
		},
	}

	rooms := []apartment.Room{
		{
			ID:   "1",
			Name: "bathroom",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2000, Y: 0},
				{X: 2000, Y: 2000},
				{X: 0, Y: 2000},
			},
			Furniture: []string{"2"},

		},
		{
			ID:   "2",
			Name: "kitchen",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
			Furniture: []string{"1"},
		},
		{
			ID:   "3",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 2000, Y: 0},
				{X: 2000, Y: 2000},
				{X: 0, Y: 2000},
			},
		},
	}

	ap := &apartment.Apartment{
		Rooms: rooms,
		Furniture: []apartment.Furniture{sink, stove},
	}

	selectedLevels := map[string]string{
		"security": "1",
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

	assert.Equal(t, 3000, priceInfo.MinPrice)
	assert.Equal(t, 7000, priceInfo.MaxPrice)
}
