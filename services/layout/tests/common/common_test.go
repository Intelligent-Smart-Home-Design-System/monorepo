package common

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
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
	room := entities.Room{
		ID:   "1",
		Name: "kitchen",
		Area: []entities.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}
	apartment := &entities.Apartment{
		Rooms: []entities.Room{room},
	}
	apartment.MakeDependency()

	selectedLevels := map[string]string{
		"security": "1",
	}

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	tracksConfig, err1 := configs.LoadTracksConfig(GetTracksPath())
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
	apartment := &entities.Apartment{
		Rooms: nil,
	}
	apartment.MakeDependency()

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
