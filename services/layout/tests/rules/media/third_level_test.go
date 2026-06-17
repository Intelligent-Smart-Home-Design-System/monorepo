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

func TestSimpleThirdLevelScript(t *testing.T) {
	sofaFurniture := apartment.Furniture{
		ID:     "1",
		Category:   apartment.Sofa,
		Points: []point.Point{
			{X: 3, Y: 5},
			{X: 6, Y: 5},
			{X: 6, Y: 4},
			{X: 3, Y: 4},
		},
	}

	walls := []apartment.Wall{
		{
			ID: "1", 
			Points: []point.Point{
				{X: 0, Y: 0}, 
				{X: 9, Y: 0},
			},
			Width: 9,
		},
		{
			ID: "2", 
			Points: []point.Point{
				{X: 9, Y: 0}, 
				{X: 9, Y: 7},
			},
			Width: 7,
		},
		{
			ID: "3", 
			Points: []point.Point{
				{X: 9, Y: 7}, 
				{X: 0, Y: 7},
			},
			Width: 9,
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

	room := apartment.Room{
		ID:   "1",
		Name: "living",
		Area: []point.Point{
			{X: 0, Y: 0},
			{X: 9, Y: 0},
			{X: 9, Y: 7},
			{X: 0, Y: 7},
		},
		AreaM2:     63,
		Furniture:  []string{"1"},
		Walls:      []string{"1", "2", "3", "4"},
	}

	ap := &apartment.Apartment{
		Rooms:      []apartment.Room{room},
		Walls:  	walls,
		Furniture: []apartment.Furniture{sofaFurniture},
	}

	selectedLevels := map[string]string{
		"media": "3",
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

	foundTV := false
	foundSpeaker := false
	foundSubwoofer := false
	for _, devicePlacement := range globalPlacement.Placements[room.ID] {
		switch devicePlacement.Device.Type {
		case "smart_tv":
			foundTV = true

			assert.Equal(t, &point.Point{X: 4.5, Y: 7.0}, devicePlacement.Position)
		case "smart_speaker":
			foundSpeaker = true

			assert.Equal(t, 7.0, devicePlacement.Position.Y)

			tvX := 4.5
			isLeft := devicePlacement.Position.X <= tvX - 0.5
			isRight := devicePlacement.Position.X >= tvX + 0.5
			assert.True(t, isLeft || isRight)
		case "subwoofer":
			foundSubwoofer = true

			isLeftCorner := devicePlacement.Position.X < 1.0
			isRightCorner := devicePlacement.Position.X > 8.0

			assert.True(t, isLeftCorner || isRightCorner)
			assert.InDelta(t, 7.0, devicePlacement.Position.Y, 1)
		}
	}
	assert.True(t, foundTV)
	assert.True(t, foundSpeaker)
	assert.True(t, foundSubwoofer)
}
