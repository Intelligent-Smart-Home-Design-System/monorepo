package household

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/tests/rules"
	"github.com/stretchr/testify/require"
)

func placeRobotVacuumOnLevel(t *testing.T, level string) (*device.Placement, *filters.RobotVacuumFilter) {
	t.Helper()

	require.NoError(t, configs.LoadTracksConfig(rules.GetTracksPath()))
	require.NoError(t, configs.LoadDevicesConfig(rules.GetDevicesPath()))

	st := storage.NewStorage()
	st.LoadAllHouseholdRules()

	e := engine.NewEngine(st)
	layout, err := e.PlaceDevices(testApartment(), map[string]string{
		"household": level,
	})

	require.NoError(t, err)
	require.Equal(t, 1, countRobotVacuumPlacements(layout))

	placement := findRobotVacuumPlacement(layout)
	require.NotNil(t, placement)
	require.NotNil(t, placement.Position)

	robotFilter, ok := placement.Filters.(*filters.RobotVacuumFilter)
	require.True(t, ok)

	return placement, robotFilter
}

func testApartment() *apartment.Apartment {
	return &apartment.Apartment{
		Walls: []apartment.Wall{
			{ID: "w1", Points: []point.Point{{X: 0, Y: 0}, {X: 4000, Y: 0}}},
			{ID: "w2", Points: []point.Point{{X: 4000, Y: 0}, {X: 4000, Y: 4000}}},
			{ID: "w3", Points: []point.Point{{X: 4000, Y: 4000}, {X: 0, Y: 4000}}},
			{ID: "w4", Points: []point.Point{{X: 0, Y: 4000}, {X: 0, Y: 0}}},
		},
		Rooms: []apartment.Room{
			{
				ID:   "r1",
				Name: apartment.RoomLiving,
				Area: []point.Point{
					{X: 0, Y: 0},
					{X: 4000, Y: 0},
					{X: 4000, Y: 4000},
					{X: 0, Y: 4000},
				},
				Walls: []string{"w1", "w2", "w3", "w4"},
			},
			{
				ID:   "r2",
				Name: apartment.RoomBathroom,
				Area: []point.Point{
					{X: 5000, Y: 0},
					{X: 7000, Y: 0},
					{X: 7000, Y: 2000},
					{X: 5000, Y: 2000},
				},
			},
		},
	}
}

func findRobotVacuumPlacement(layout *apartment.Layout) *device.Placement {
	for _, roomPlacements := range layout.Placements {
		for _, placement := range roomPlacements {
			if placement.Device.Type == "robot_vacuum" {
				return placement
			}
		}
	}

	return nil
}

func countRobotVacuumPlacements(layout *apartment.Layout) int {
	count := 0

	for _, roomPlacements := range layout.Placements {
		for _, placement := range roomPlacements {
			if placement.Device.Type == "robot_vacuum" {
				count++
			}
		}
	}

	return count
}
