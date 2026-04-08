package rules

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/stretchr/testify/assert"
)

func GetDevicesPath() string {
	return "../../internal/configs/devices.json"
}

func GetTracksPath() string {
	return "../../internal/configs/tracks.json"
}

func GetSimpleTracksPath() string {
	return "../testdata/simple_tracks.json"
}

func TestLoadTracksConfig(t *testing.T) {
	_, err := configs.LoadTracksConfig(GetTracksPath())

	assert.NoError(t, err)
}

func TestLoadDevicesConfig(t *testing.T) {
	_, err := configs.LoadTracksConfig(GetDevicesPath())

	assert.NoError(t, err)
}

func TestLaunch(t *testing.T) {
	room := &apartment.Room{
		ID:        "1",
		Name:      "kitchen",
		WetPoints: make([]*device.Point, 0),
	}
	apartment := &apartment.Apartment{
		ID:     "1",
		Tracks: []string{"security"},
		Rooms:  []*apartment.Room{room},
	}
	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	_, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.NoError(t, err)
}

func TestNilApartment(t *testing.T) {
	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	_, err := engine.PlaceDevices(nil, selectedLevels)

	assert.Error(t, err)
}

func TestNilRoomsStruct(t *testing.T) {
	apartment := &apartment.Apartment{
		ID:     "1",
		Tracks: []string{"security"},
		Rooms:  nil,
	}

	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	_, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.Error(t, err)
}

func TestSimpleScript(t *testing.T) {
	room := &apartment.Room{
		ID:        "1",
		Name:      "bathroom",
		WetPoints: []*device.Point{{X: 1, Y: 2, Z: 0}},
	}
	apartment := &apartment.Apartment{
		ID:     "1",
		Tracks: []string{"security"},
		Rooms:  []*apartment.Room{room},
	}

	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	globalPlacement, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements[room.ID] {
		assert.Equal(t, "water_leak_sensor", devicePlacement.Device.Type)
		assert.Equal(t, device.Point{X: 1, Y: 2, Z: 0}, *devicePlacement.Place)
	}
}

func TestMultipleRoomsOneWetPoint(t *testing.T) {
	rooms := []*apartment.Room{
		{
			ID:        "1",
			Name:      "bathroom",
			WetPoints: []*device.Point{{X: 1, Y: 2, Z: 0}},
		},
		{
			ID:        "2",
			Name:      "kitchen",
			WetPoints: make([]*device.Point, 0),
		},
	}
	apartment := &apartment.Apartment{
		ID:     "1",
		Tracks: []string{"security"},
		Rooms:  rooms,
	}

	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	globalPlacement, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements[rooms[0].ID] {
		assert.Equal(t, "water_leak_sensor", devicePlacement.Device.Type)
		assert.Equal(t, device.Point{X: 1, Y: 2, Z: 0}, *devicePlacement.Place)
	}

	assert.Equal(t, 0, len(globalPlacement.Placements[rooms[1].ID]))
}

func TestMultipleRoomsMultipleWetPoints(t *testing.T) {
	rooms := []*apartment.Room{
		{
			ID:        "1",
			Name:      "bathroom",
			WetPoints: []*device.Point{{X: 1, Y: 2, Z: 0}},
		},
		{
			ID:        "2",
			Name:      "kitchen",
			WetPoints: []*device.Point{{X: 5, Y: 10, Z: 0}},
		},
	}
	apartment := &apartment.Apartment{
		ID:     "1",
		Tracks: []string{"security"},
		Rooms:  rooms,
	}

	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	globalPlacement, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements[rooms[0].ID] {
		assert.Equal(t, "water_leak_sensor", devicePlacement.Device.Type)
		assert.Equal(t, device.Point{X: 1, Y: 2, Z: 0}, *devicePlacement.Place)
	}

	assert.Equal(t, 1, len(globalPlacement.Placements[rooms[0].ID]))

	for _, devicePlacement := range globalPlacement.Placements[rooms[1].ID] {
		assert.Equal(t, "water_leak_sensor", devicePlacement.Device.Type)
		assert.Equal(t, device.Point{X: 5, Y: 10, Z: 0}, *devicePlacement.Place)
	}

	assert.Equal(t, 1, len(globalPlacement.Placements[rooms[1].ID]))
}

func TestPriceCalculation(t *testing.T) {
	rooms := []*apartment.Room{
		{
			ID:        "1",
			Name:      "bathroom",
			WetPoints: []*device.Point{{X: 1, Y: 2, Z: 0}},
		},
		{
			ID:        "2",
			Name:      "kitchen",
			WetPoints: []*device.Point{{X: 5, Y: 10, Z: 0}},
		},
	}
	apartment := &apartment.Apartment{
		ID:     "1",
		Tracks: []string{"security"},
		Rooms:  rooms,
	}

	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	tracksConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	devicesConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)
	globalPlacement, err := engine.PlaceDevices(apartment, selectedLevels)

	assert.NoError(t, err)

	priceInfo := engine.CalcLayoutPrice(globalPlacement)

	assert.Equal(t, 2000, priceInfo.MinPrice)
	assert.Equal(t, 4000, priceInfo.MaxPrice)
}
