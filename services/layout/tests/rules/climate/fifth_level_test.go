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

func TestClimateLevel5(t *testing.T) {
	apartmentStruct := &apartment.Apartment{
		Walls: []apartment.Wall{
			{ID: "w1", Points: []point.Point{{X: 0, Y: 0}, {X: 4000, Y: 0}}},
			{ID: "w2", Points: []point.Point{{X: 4000, Y: 0}, {X: 4000, Y: 4}}},
			{ID: "w3", Points: []point.Point{{X: 4000, Y: 4000}, {X: 0, Y: 4000}}},
			{ID: "w4", Points: []point.Point{{X: 0, Y: 4000}, {X: 0, Y: 0}}},
			{ID: "w5", Points: []point.Point{{X: 5000, Y: 0}, {X: 9000, Y: 0}}},
			{ID: "w6", Points: []point.Point{{X: 9000, Y: 0}, {X: 9000, Y: 4000}}},
			{ID: "w7", Points: []point.Point{{X: 9000, Y: 4000}, {X: 5000, Y: 4000}}},
			{ID: "w8", Points: []point.Point{{X: 5000, Y: 4000}, {X: 5000, Y: 0}}},
			{ID: "w9", Points: []point.Point{{X: 0, Y: 5000}, {X: 4000, Y: 5000}}},
			{ID: "w10", Points: []point.Point{{X: 4000, Y: 5000}, {X: 4000, Y: 9000}}},
			{ID: "w11", Points: []point.Point{{X: 4000, Y: 9000}, {X: 0, Y: 9000}}},
			{ID: "w12", Points: []point.Point{{X: 0, Y: 9000}, {X: 0, Y: 5000}}},
			{ID: "w13", Points: []point.Point{{X: 5000, Y: 5000}, {X: 9000, Y: 5000}}},
			{ID: "w14", Points: []point.Point{{X: 9000, Y: 5000}, {X: 9000, Y: 9000}}},
			{ID: "w15", Points: []point.Point{{X: 9000, Y: 9000}, {X: 5000, Y: 9000}}},
			{ID: "w16", Points: []point.Point{{X: 5000, Y: 9000}, {X: 5000, Y: 5000}}},
			{ID: "w17", Points: []point.Point{{X: 10000, Y: 0}, {X: 14000, Y: 0}}},
			{ID: "w18", Points: []point.Point{{X: 14000, Y: 0}, {X: 14000, Y: 4}}},
			{ID: "w19", Points: []point.Point{{X: 14000, Y: 4000}, {X: 10000, Y: 4000}}},
			{ID: "w20", Points: []point.Point{{X: 10000, Y: 4000}, {X: 10000, Y: 0}}},
		},
		Rooms: []apartment.Room{
			{ID: "r1", Name: apartment.RoomLiving, AreaM2: 16, Area: []point.Point{{X: 0, Y: 0}, {X: 4000, Y: 0}, {X: 4000, Y: 4000}, {X: 0, Y: 4000}}, Walls: []string{"w1", "w2", "w3", "w4"}},
			{ID: "r2", Name: apartment.RoomBedroom, AreaM2: 16, Area: []point.Point{{X: 5000, Y: 0}, {X: 9000, Y: 0}, {X: 9000, Y: 4000}, {X: 5000, Y: 4000}}, Walls: []string{"w5", "w6", "w7", "w8"}},
			{ID: "r3", Name: apartment.RoomKitchen, AreaM2: 16, Area: []point.Point{{X: 0, Y: 5000}, {X: 4000, Y: 5000}, {X: 4000, Y: 9000}, {X: 0, Y: 9000}}, Walls: []string{"w9", "w10", "w11", "w12"}},
			{ID: "r4", Name: apartment.RoomCabinet, AreaM2: 16, Area: []point.Point{{X: 5000, Y: 5000}, {X: 9000, Y: 5000}, {X: 9000, Y: 9000}, {X: 5000, Y: 9000}}, Walls: []string{"w13", "w14", "w15", "w16"}},
			{ID: "r5", Name: apartment.RoomBathroom, AreaM2: 16, Area: []point.Point{{X: 10000, Y: 0}, {X: 14000, Y: 0}, {X: 14000, Y: 4000}, {X: 10000, Y: 4000}}, Walls: []string{"w17", "w18", "w19", "w20"}},
		},
	}

	st := storage.NewStorage()
	st.LoadAllClimateRules()
	assert.NoError(t, configs.LoadTracksConfig(rules.GetTracksPath()))
	assert.NoError(t, configs.LoadDevicesConfig(rules.GetDevicesPath()))

	e := engine.NewEngine(st)
	globalPlacement, err := e.PlaceDevices(apartmentStruct, map[string]string{"climate": "5"})
	assert.NoError(t, err)

	for _, roomID := range []string{"r1", "r2", "r4"} {
		assert.True(t, globalPlacement.HasDeviceInRoom("air_conditioner", roomID))
	}
	assert.False(t, globalPlacement.HasDeviceInRoom("air_conditioner", "r3"))
	assert.False(t, globalPlacement.HasDeviceInRoom("air_conditioner", "r5"))
}
