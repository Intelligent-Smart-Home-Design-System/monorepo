package household

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHouseholdLevel1(t *testing.T) {
	placement, robotFilter := placeRobotVacuumOnLevel(t, "1")

	assert.Equal(t, "robot_vacuum", placement.Device.Type)
	assert.Equal(t, "household", placement.Device.Track)

	assert.Equal(t, float64(75), robotFilter.NoiseLevelDB)
	assert.Equal(t, 2000, robotFilter.SuctionPowerPA)
	assert.Equal(t, "basic", robotFilter.NavigationType)
	assert.False(t, robotFilter.RoomMapping)
	assert.False(t, robotFilter.WetCleaning)
	assert.False(t, robotFilter.CarpetDetection)
	assert.False(t, robotFilter.ObstacleAvoidance)
	assert.False(t, robotFilter.AutoEmptyStation)
	assert.False(t, robotFilter.VoiceAssistantSupport)
}