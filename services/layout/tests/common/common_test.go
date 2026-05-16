package common

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/stretchr/testify/assert"
)

func TestLoadTracksConfig(t *testing.T) {
	_, err := configs.LoadTracksConfig(GetTracksPath())

	assert.NoError(t, err)
}

func TestLoadDevicesConfig(t *testing.T) {
	_, err := configs.LoadTracksConfig(GetDevicesPath())

	assert.NoError(t, err)
}

func TestLaunch(t *testing.T) {
	room := apartment.Room{
		ID:   "1",
		Name: "kitchen",
		Area: []point.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}
	apartmentStruct := &apartment.Apartment{
		Rooms: []apartment.Room{room},
	}

	selectedLevels := map[string]string{
		"security": "1",
	}

	trackConfig, err1 := configs.LoadTracksConfig(GetTracksPath())
	deviceConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules(deviceConfig)

	engine := engine.NewEngine(storage, trackConfig, deviceConfig)
	_, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

	assert.NoError(t, err)
}

func TestNilApartment(t *testing.T) {
	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	trackConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	deviceConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, trackConfig, deviceConfig)
	_, err := engine.PlaceDevices(nil, selectedLevels)

	assert.Error(t, err)
}

func TestNilRoomsStruct(t *testing.T) {
	apartmentStruct := &apartment.Apartment{
		Rooms: nil,
	}

	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	trackConfig, err1 := configs.LoadTracksConfig(GetSimpleTracksPath())
	deviceConfig, err2 := configs.LoadDevicesConfig(GetDevicesPath())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	engine := engine.NewEngine(storage, trackConfig, deviceConfig)
	_, err := engine.PlaceDevices(apartmentStruct, selectedLevels)

	assert.Error(t, err)
}
