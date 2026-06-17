package media

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
	window := apartment.Window{
		ID:     "1",
		Points: []point.Point{
			{X: 3000, Y: 0},
			{X: 5000, Y: 0},
		},
		Width:  2000,
		Room:  "livingroom",
	}

	walls := []apartment.Wall{
		{
			ID: "1", 
			Points: []point.Point{
				{X: 0, Y: 0}, 
				{X: 8000, Y: 0},
			},
			Width: 8,
		},
		{
			ID: "2", 
			Points: []point.Point{
				{X: 8000, Y: 0}, 
				{X: 8000, Y: 6000},
			},
			Width: 6000,
		},
		{
			ID: "3", 
			Points: []point.Point{
				{X: 8000, Y: 6000},
				{X: 0, Y: 6000},
			},
			Width: 8000,
		},
		{
			ID: "4", 
			Points: []point.Point{
				{X: 0, Y: 6000}, 
				{X: 0, Y: 0},
			},
			Width: 6000,
		},
	}

	room := apartment.Room{
		ID:   "1",
		Name: "living",
		Area: []point.Point{
			{X: 0, Y: 0},
			{X: 8000, Y: 0},
			{X: 8000, Y: 6000},
			{X: 0, Y: 6000},
		},
		Windows:   []string{"1"},
		Walls:     []string{"1", "2", "3", "4"},
	}

	ap := &apartment.Apartment{
		Rooms:   []apartment.Room{room},
		Walls:   walls,
		Windows: []apartment.Window{window},
	}

	selectedLevels := map[string]string{
		"media": "1",
	}

	err1 := configs.LoadTracksConfig(rules.GetTracksPath())
	err2 := configs.LoadDevicesConfig(rules.GetDevicesPath())
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	storage := storage.NewStorage()
	storage.LoadAllMediaRules()

	engine := engine.NewEngine(storage)
	globalPlacement, err := engine.PlaceDevices(ap, selectedLevels)

	assert.NoError(t, err)

	foundSpeaker := false
	for _, devicePlacement := range globalPlacement.Placements[room.ID] {
		if devicePlacement.Device.Type == "smart_speaker" {
			foundSpeaker = true

			assert.Equal(t, 0.0, devicePlacement.Position.Y)
			assert.GreaterOrEqual(t, devicePlacement.Position.X, 3000.0)
			assert.LessOrEqual(t, devicePlacement.Position.X, 5000.0)
		}
	}
	assert.True(t, foundSpeaker)
}