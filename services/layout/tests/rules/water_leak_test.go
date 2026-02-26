package rules

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/security"
	"github.com/stretchr/testify/assert"
)

func TestLaunch(t *testing.T) {
	room := &entities.Room{
		ID: "1", 
		Name: "kitchen", 
		WetPoints: make([]*entities.Point, 0),
	}
	
	apartment := &entities.Apartment{ID: "1", Tracks: []string{"security"}, Rooms: []*entities.Room{room}}

	waterLeakRule := security.NewWaterLeakRule("1", "security")
	device_rules := []rules.Rule{waterLeakRule}
	
	engine := engine.NewEngine(device_rules)
	_, err := engine.PlaceDevices(apartment)

	assert.NoError(t, err)
}

func TestNilApartment(t *testing.T) {
	waterLeakRule := security.NewWaterLeakRule("1", "security")
	device_rules := []rules.Rule{waterLeakRule}
	
	engine := engine.NewEngine(device_rules)
	_, err := engine.PlaceDevices(nil)

	assert.Error(t, err)
}

func TestNilRoomsStruct(t *testing.T) {
	apartment := &entities.Apartment{ID: "1", Tracks: []string{"security"}, Rooms: nil}

	waterLeakRule := security.NewWaterLeakRule("1", "security")
	device_rules := []rules.Rule{waterLeakRule}
	
	engine := engine.NewEngine(device_rules)
	_, err := engine.PlaceDevices(apartment)

	assert.Error(t, err)
}

func TestNilRoom(t *testing.T) {
	room := &entities.Room{
		ID: "1", 
		Name: "kitchen", 
		WetPoints: make([]*entities.Point, 0),
	}
	apartment := &entities.Apartment{ID: "1", Tracks: []string{"security"}, Rooms: []*entities.Room{room, nil}}

	waterLeakRule := security.NewWaterLeakRule("1", "security")
	device_rules := []rules.Rule{waterLeakRule}
	
	engine := engine.NewEngine(device_rules)
	_, err := engine.PlaceDevices(apartment)

	assert.Error(t, err)
}

func TestSimpleScript(t *testing.T) {
	room := &entities.Room{
		ID: "1", 
		Name: "bathroom", 
		WetPoints: []*entities.Point{{X: 1, Y: 2, Z: 0}},
	}
	apartment := &entities.Apartment{ID: "1", Tracks: []string{"security"}, Rooms: []*entities.Room{room}}

	waterLeakRule := security.NewWaterLeakRule("1", "security")
	device_rules := []rules.Rule{waterLeakRule}
	
	engine := engine.NewEngine(device_rules)
	globalPlacement, err := engine.PlaceDevices(apartment)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements[room.ID] {
		assert.Equal(t, "water_leak", devicePlacement.Device.Type)
		assert.Equal(t, entities.Point{X: 1, Y: 2, Z: 0}, *devicePlacement.Place)
	}
}

func TestMultipleRoomsOneWetPoint(t *testing.T) {
	rooms := []*entities.Room{
		{
			ID: "1", 
			Name: "bathroom", 
			WetPoints: []*entities.Point{{X: 1, Y: 2, Z: 0}},
		},
		{
			ID: "2", 
			Name: "kitchen", 
			WetPoints: make([]*entities.Point, 0),
		},
	}
	apartment := &entities.Apartment{ID: "1", Tracks: []string{"security"}, Rooms: rooms}

	waterLeakRule := security.NewWaterLeakRule("1", "security")
	device_rules := []rules.Rule{waterLeakRule}
	
	engine := engine.NewEngine(device_rules)
	globalPlacement, err := engine.PlaceDevices(apartment)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements[rooms[0].ID] {
		assert.Equal(t, "water_leak", devicePlacement.Device.Type)
		assert.Equal(t, entities.Point{X: 1, Y: 2, Z: 0}, *devicePlacement.Place)
	}
	
	assert.Equal(t, 0, len(globalPlacement.Placements[rooms[1].ID]))
}

func TestMultipleRoomsMultipleWetPoints(t *testing.T) {
	rooms := []*entities.Room{
		{
			ID: "1", 
			Name: "bathroom", 
			WetPoints: []*entities.Point{{X: 1, Y: 2, Z: 0}},
		},
		{
			ID: "2", 
			Name: "kitchen", 
			WetPoints: []*entities.Point{{X: 5, Y: 10, Z: 0}},
		},
	}
	apartment := &entities.Apartment{ID: "1", Tracks: []string{"security"}, Rooms: rooms}

	waterLeakRule := security.NewWaterLeakRule("1", "security")
	device_rules := []rules.Rule{waterLeakRule}
	
	engine := engine.NewEngine(device_rules)
	globalPlacement, err := engine.PlaceDevices(apartment)

	assert.NoError(t, err)

	for _, devicePlacement := range globalPlacement.Placements[rooms[0].ID] {
		assert.Equal(t, "water_leak", devicePlacement.Device.Type)
		assert.Equal(t, entities.Point{X: 1, Y: 2, Z: 0}, *devicePlacement.Place)
	}

	assert.Equal(t, 1, len(globalPlacement.Placements[rooms[0].ID]))
	
	for _, devicePlacement := range globalPlacement.Placements[rooms[1].ID] {
		assert.Equal(t, "water_leak", devicePlacement.Device.Type)
		assert.Equal(t, entities.Point{X: 5, Y: 10, Z: 0}, *devicePlacement.Place)
	}

	assert.Equal(t, 1, len(globalPlacement.Placements[rooms[1].ID]))
}
