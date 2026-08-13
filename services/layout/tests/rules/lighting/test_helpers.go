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
        {ID: "r1", Name: apartment.RoomLiving, Area: []point.Point{{X: 0, Y: 0}, {X: 4000, Y: 0}, {X: 4000, Y: 4000}, {X: 0, Y: 4000}}},
        {ID: "r2", Name: apartment.RoomKitchen, Area: []point.Point{{X: 5000, Y: 0}, {X: 8000, Y: 0}, {X: 8000, Y: 3000}, {X: 5000, Y: 3000}}},
        {ID: "r3", Name: apartment.RoomPassage, Area: []point.Point{{X: 0, Y: 5000}, {X: 7000, Y: 5000}, {X: 7000, Y: 6000}, {X: 0, Y: 6000}}},
        {ID: "r4", Name: apartment.RoomBathroom, Area: []point.Point{{X: 8000, Y: 4000}, {X: 10000, Y: 4000}, {X: 10000, Y: 6000}, {X: 8000, Y: 6000}}},
        {ID: "r5", Name: apartment.RoomBedroom, Area: []point.Point{{X: 0, Y: 7000}, {X: 4000, Y: 7000}, {X: 4000, Y: 10000}, {X: 0, Y: 10000}}},
        {ID: "r6", Name: apartment.RoomCabinet, Area: []point.Point{{X: 5000, Y: 7000}, {X: 8000, Y: 7000}, {X: 8000, Y: 10000}, {X: 5000, Y: 10000}}},
    }

    windows := []apartment.Window{
        {ID: "w1", Points: []point.Point{{X: 0, Y: 1000}, {X: 0, Y: 2000}}, Room: apartment.RoomLiving},
        {ID: "w2", Points: []point.Point{{X: 8000, Y: 1000}, {X: 8000, Y: 2000}}, Room: apartment.RoomKitchen},
        {ID: "w3", Points: []point.Point{{X: 0, Y: 8000}, {X: 0, Y: 9000}}, Room: apartment.RoomBedroom},
    }

    doors := []apartment.Door{
        {ID: "d1", Points: []point.Point{{X: 2000, Y: 4000}, {X: 3000, Y: 4000}}, Rooms: []string{"r1", "r3"}},
        {ID: "d2", Points: []point.Point{{X: 6000, Y: 3000}, {X: 7000, Y: 3000}}, Rooms: []string{"r2", "r3"}},
        {ID: "d3", Points: []point.Point{{X: 8000, Y: 5000}, {X: 8000, Y: 5500}}, Rooms: []string{"r3", "r4"}},
        {ID: "d4", Points: []point.Point{{X: 2000, Y: 7000}, {X: 3000, Y: 7000}}, Rooms: []string{"r3", "r5"}},
        {ID: "d5", Points: []point.Point{{X: 5500, Y: 7000}, {X: 6500, Y: 7000}}, Rooms: []string{"r3", "r6"}},
    }

    return &apartment.Apartment{Rooms: rooms, Windows: windows, Doors: doors}
}
