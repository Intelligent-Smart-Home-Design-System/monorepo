package rules

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/stretchr/testify/assert"
)

// func mmToM(val float64) float64 {
// 	return val / 1000.0
// }

// func convertPoints(points [][2]float64) []point.Point {
// 	res := make([]point.Point, len(points))
// 	for i, p := range points {
// 		res[i] = point.Point{X: mmToM(p[0]), Y: mmToM(p[1])}
// 	}
// 	return res
// }

func TestComplexApartmentPlacement(t *testing.T) {
	err1 := configs.LoadTracksConfig("../../internal/configs/tracks.json")
	err2 := configs.LoadDevicesConfig("../../internal/configs/devices.json")
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	data, err := os.ReadFile("../../../floor-parser/data/apartment_third_floor_insert_blocks.expected.json")
	assert.NoError(t, err)

	ap := &apartment.Apartment{}
	err = json.Unmarshal(data, &ap)
	assert.NoError(t, err)

	storage := storage.NewStorage()
	storage.LoadAllRules()

	eng := engine.NewEngine(storage)
	selectedLevels := map[string]string{
		"security": "5",
		"media":    "3",
	}

	layout, err := eng.PlaceDevices(ap, selectedLevels)
	assert.NoError(t, err)
	assert.NotNil(t, layout)

	totalPlacements := 0
	for _, placements := range layout.Placements {
		totalPlacements += len(placements)
	}
	assert.Greater(t, totalPlacements, 10)
}