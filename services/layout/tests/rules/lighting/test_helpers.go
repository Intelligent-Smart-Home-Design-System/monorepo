package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
)

type levelPrice struct {
	expectedMin int
	expectedMax int
	actualMin   int
	actualMax   int
}

func placeLightingLevel(apartmentStruct *apartment.Apartment, level string) (*apartment.Layout, *configs.Devices, error) {
	selectedLevels := map[string]string{"lighting": level}

	st := storage.NewStorage()
	st.LoadAllLightingRules()

	err1 := configs.LoadTracksConfig("../../../internal/configs/tracks.json")
	err2 := configs.LoadDevicesConfig("../../../internal/configs/devices.json")
	if err1 != nil {
		return nil, nil, err1
	}
	if err2 != nil {
		return nil, nil, err2
	}

	devicesConfig := configs.GetGlobalDevicesConfig()

	e := engine.NewEngine(st)
	layout, err := e.PlaceDevices(apartmentStruct, selectedLevels)
	if err != nil {
		return nil, nil, err
	}

	return layout, devicesConfig, nil
}

func countDeviceType(layout *apartment.Layout, deviceType string) int {
	count := 0
	for _, roomPlacements := range layout.Placements {
		for _, placement := range roomPlacements {
			if placement.Device.Type == deviceType {
				count++
			}
		}
	}
	return count
}

func calculateExpectedAndActualPrice(layout *apartment.Layout, devicesConfig *configs.Devices) levelPrice {
	expectedMin := 0
	expectedMax := 0

	for _, roomPlacements := range layout.Placements {
		for _, placement := range roomPlacements {
			cfg := devicesConfig.Devices[placement.Device.Type]
			expectedMin += cfg.Price.Min
			expectedMax += cfg.Price.Max
		}
	}

	st := storage.NewStorage()
	st.LoadAllLightingRules()
	_ = configs.LoadTracksConfig("../../../internal/configs/tracks.json")
	e := engine.NewEngine(st)
	actual := e.CalculateLayoutPrice(layout)

	return levelPrice{
		expectedMin: expectedMin,
		expectedMax: expectedMax,
		actualMin:   actual.MinPrice,
		actualMax:   actual.MaxPrice,
	}
}

func buildLightingApartmentForHighLevels() *apartment.Apartment {
	rooms := []apartment.Room{
		{ID: "r1", Name: apartment.RoomLiving, Area: []point.Point{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}},
		{ID: "r2", Name: apartment.RoomKitchen, Area: []point.Point{{X: 5, Y: 0}, {X: 8, Y: 0}, {X: 8, Y: 3}, {X: 5, Y: 3}}},
		{ID: "r3", Name: apartment.RoomPassage, Area: []point.Point{{X: 0, Y: 5}, {X: 7, Y: 5}, {X: 7, Y: 6}, {X: 0, Y: 6}}},
		{ID: "r4", Name: apartment.RoomBathroom, Area: []point.Point{{X: 8, Y: 4}, {X: 10, Y: 4}, {X: 10, Y: 6}, {X: 8, Y: 6}}},
		{ID: "r5", Name: apartment.RoomBedroom, Area: []point.Point{{X: 0, Y: 7}, {X: 4, Y: 7}, {X: 4, Y: 10}, {X: 0, Y: 10}}},
		{ID: "r6", Name: apartment.RoomCabinet, Area: []point.Point{{X: 5, Y: 7}, {X: 8, Y: 7}, {X: 8, Y: 10}, {X: 5, Y: 10}}},
	}

	windows := []apartment.Window{
		{ID: "w1", Points: []point.Point{{X: 0, Y: 1}, {X: 0, Y: 2}}, Rooms: []string{"r1"}},
		{ID: "w2", Points: []point.Point{{X: 8, Y: 1}, {X: 8, Y: 2}}, Rooms: []string{"r2"}},
		{ID: "w3", Points: []point.Point{{X: 0, Y: 8}, {X: 0, Y: 9}}, Rooms: []string{"r5"}},
	}

	doors := []apartment.Door{
		{ID: "d1", Points: []point.Point{{X: 2, Y: 4}, {X: 3, Y: 4}}, Rooms: []string{"r1", "r3"}},
		{ID: "d2", Points: []point.Point{{X: 6, Y: 3}, {X: 7, Y: 3}}, Rooms: []string{"r2", "r3"}},
		{ID: "d3", Points: []point.Point{{X: 8, Y: 5}, {X: 8, Y: 5.5}}, Rooms: []string{"r3", "r4"}},
		{ID: "d4", Points: []point.Point{{X: 2, Y: 7}, {X: 3, Y: 7}}, Rooms: []string{"r3", "r5"}},
		{ID: "d5", Points: []point.Point{{X: 5.5, Y: 7}, {X: 6.5, Y: 7}}, Rooms: []string{"r3", "r6"}},
	}

	return &apartment.Apartment{Rooms: rooms, Windows: windows, Doors: doors}
}
