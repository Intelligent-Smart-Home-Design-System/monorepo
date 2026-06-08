package household

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHouseholdLevel3(t *testing.T) {
	_, robotFilter := placeRobotVacuumOnLevel(t, "3")

	assert.Equal(t, float64(70), robotFilter.NoiseLevelDB)
	assert.Equal(t, 5000, robotFilter.SuctionPowerPA)
	assert.Equal(t, "lidar", robotFilter.NavigationType)
	assert.True(t, robotFilter.RoomMapping)
	assert.True(t, robotFilter.WetCleaning)
	assert.False(t, robotFilter.CarpetDetection)
	assert.True(t, robotFilter.ObstacleAvoidance)
	assert.False(t, robotFilter.AutoEmptyStation)
	assert.False(t, robotFilter.VoiceAssistantSupport)
}