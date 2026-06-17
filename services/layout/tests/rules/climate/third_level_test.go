package climate

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

func TestClimateLevel3(t *testing.T) {
	window := apartment.Window{ID: "win1", Points: []point.Point{{X: 0, Y: 1}, {X: 0, Y: 2}}, Width: 1, Rooms: []string{"r1"}}
	apartmentStruct := &apartment.Apartment{
		Windows: []apartment.Window{window},
		Walls: []apartment.Wall{
			{ID: "w1", Points: []point.Point{{X: 0, Y: 0}, {X: 4, Y: 0}}},
			{ID: "w2", Points: []point.Point{{X: 4, Y: 0}, {X: 4, Y: 4}}},
			{ID: "w3", Points: []point.Point{{X: 4, Y: 4}, {X: 0, Y: 4}}},
			{ID: "w4", Points: []point.Point{{X: 0, Y: 4}, {X: 0, Y: 0}}},
			{ID: "w5", Points: []point.Point{{X: 5, Y: 0}, {X: 9, Y: 0}}},
			{ID: "w6", Points: []point.Point{{X: 9, Y: 0}, {X: 9, Y: 4}}},
			{ID: "w7", Points: []point.Point{{X: 9, Y: 4}, {X: 5, Y: 4}}},
			{ID: "w8", Points: []point.Point{{X: 5, Y: 4}, {X: 5, Y: 0}}},
			{ID: "w9", Points: []point.Point{{X: 0, Y: 5}, {X: 4, Y: 5}}},
			{ID: "w10", Points: []point.Point{{X: 4, Y: 5}, {X: 4, Y: 9}}},
			{ID: "w11", Points: []point.Point{{X: 4, Y: 9}, {X: 0, Y: 9}}},
			{ID: "w12", Points: []point.Point{{X: 0, Y: 9}, {X: 0, Y: 5}}},
			{ID: "w13", Points: []point.Point{{X: 5, Y: 5}, {X: 9, Y: 5}}},
			{ID: "w14", Points: []point.Point{{X: 9, Y: 5}, {X: 9, Y: 9}}},
			{ID: "w15", Points: []point.Point{{X: 9, Y: 9}, {X: 5, Y: 9}}},
			{ID: "w16", Points: []point.Point{{X: 5, Y: 9}, {X: 5, Y: 5}}},
			{ID: "w17", Points: []point.Point{{X: 10, Y: 0}, {X: 14, Y: 0}}},
			{ID: "w18", Points: []point.Point{{X: 14, Y: 0}, {X: 14, Y: 4}}},
			{ID: "w19", Points: []point.Point{{X: 14, Y: 4}, {X: 10, Y: 4}}},
			{ID: "w20", Points: []point.Point{{X: 10, Y: 4}, {X: 10, Y: 0}}},
		},
		Rooms: []apartment.Room{
			{ID: "r1", Name: apartment.RoomLiving, Area: []point.Point{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, Walls: []string{"w1", "w2", "w3", "w4"}, Windows: []string{"win1"}},
			{ID: "r2", Name: apartment.RoomBedroom, Area: []point.Point{{X: 5, Y: 0}, {X: 9, Y: 0}, {X: 9, Y: 4}, {X: 5, Y: 4}}, Walls: []string{"w5", "w6", "w7", "w8"}},
			{ID: "r3", Name: apartment.RoomKitchen, Area: []point.Point{{X: 0, Y: 5}, {X: 4, Y: 5}, {X: 4, Y: 9}, {X: 0, Y: 9}}, Walls: []string{"w9", "w10", "w11", "w12"}},
			{ID: "r4", Name: apartment.RoomCabinet, Area: []point.Point{{X: 5, Y: 5}, {X: 9, Y: 5}, {X: 9, Y: 9}, {X: 5, Y: 9}}, Walls: []string{"w13", "w14", "w15", "w16"}},
			{ID: "r5", Name: apartment.RoomBathroom, Area: []point.Point{{X: 10, Y: 0}, {X: 14, Y: 0}, {X: 14, Y: 4}, {X: 10, Y: 4}}, Walls: []string{"w17", "w18", "w19", "w20"}},
		},
	}

	st := storage.NewStorage()
	st.LoadAllClimateRules()
	assert.NoError(t, configs.LoadTracksConfig(rules.GetTracksPath()))
	assert.NoError(t, configs.LoadDevicesConfig(rules.GetDevicesPath()))

	e := engine.NewEngine(st)
	globalPlacement, err := e.PlaceDevices(apartmentStruct, map[string]string{"climate": "3"})
	assert.NoError(t, err)

	for _, roomID := range []string{"r1", "r2", "r4"} {
		assert.True(t, globalPlacement.HasDeviceInRoom("co2_sensor", roomID))
	}
	for _, roomID := range []string{"r1", "r2"} {
		assert.True(t, globalPlacement.HasDeviceInRoom("air_purifier", roomID))
	}
	assert.False(t, globalPlacement.HasDeviceInRoom("air_purifier", "r4"))
}
