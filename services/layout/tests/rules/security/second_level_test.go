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

func TestSecondLevelSimpleScript(t *testing.T) {
	door := apartment.Door{
		ID: "1",
		Points: []point.Point{
			{X: 1000, Y: 0},
			{X: 2000, Y: 0},
		},
		Rooms: []string{"1"},
	}

	room := apartment.Room{
		ID:   "1",
		Name: "hall",
		Area: []point.Point{
			{X: 0, Y: 0},
			{X: 3000, Y: 0},
			{X: 3000, Y: 3000},
			{X: 0, Y: 3000},
		},
		Doors: []string{door.ID},
	}

	apartmentStruct := &apartment.Apartment{
		Doors: []apartment.Door{door},
		Rooms: []apartment.Room{room},
	}

	selectedLevels := map[string]string{
		"security": "2",
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

	for _, devicePlacement := range globalPlacement.Placements[room.ID] {
		switch devicePlacement.Device.Type {
		case "smart_lock":
			assert.Equal(t, &point.Point{X: 2000, Y: 0}, devicePlacement.Position)
		case "smart_doorbell":
			assert.Equal(t, &point.Point{X: 1000, Y: 0}, devicePlacement.Position)
		}
	}

	hallRoomKeys := make([]string, 0, len(globalPlacement.Placements["1"]))
	for _, placement := range globalPlacement.Placements["1"] {
		hallRoomKeys = append(hallRoomKeys, placement.Device.Type)
	}

	correctHallRoomKeys := []string{"smart_lock", "smart_doorbell"}
	for _, key := range correctHallRoomKeys {
		assert.Contains(t, hallRoomKeys, key)
	}

	assert.Equal(t, len(correctHallRoomKeys), len(hallRoomKeys))
}

func TestSecondLevelPriceCalculation(t *testing.T) {
	door := apartment.Door{
		ID: "1",
		Points: []point.Point{
			{X: 1000, Y: 0},
			{X: 2000, Y: 0},
		},
		Rooms: []string{"3"},
	}

	bathroomSink := apartment.Furniture{
		ID: "1",
		Category: apartment.Sink,
		Points: []point.Point{
			{X: 0, Y: 0},
			{X: 1000, Y: 0},
			{X: 1000, Y: 1000},
			{X: 0, Y: 1000},
		},
		Room: "1",
	}

	kitchenSink := apartment.Furniture{
		ID: "2",
		Category: apartment.Sink,
		Points: []point.Point{
			{X: 1000, Y: 2000},
			{X: 1000, Y: 0},
			{X: 1000, Y: 1000},
			{X: 0, Y: 1000},
		},
		Room: "2",
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
			Furniture: []string{"1"},
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
			Furniture: []string{"2"},
		},
		{
			ID:   "3",
			Name: "hall",
			Area: []point.Point{
				{X: 0, Y: 0},
				{X: 3000, Y: 0},
				{X: 3000, Y: 3000},
				{X: 0, Y: 3000},
			},
			Doors: []string{"1"},
		},
	}

	ap := &apartment.Apartment{
		Doors: []apartment.Door{door},
		Rooms: rooms,
		Furniture: []apartment.Furniture{kitchenSink, bathroomSink},
	}

	selectedLevels := map[string]string{
		"security": "2",
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

	assert.Equal(t, 29000, priceInfo.MinPrice)
	assert.Equal(t, 35000, priceInfo.MaxPrice)
}
