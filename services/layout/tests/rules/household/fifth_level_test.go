package household

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHouseholdLevel5(t *testing.T) {
	_, robotFilter := placeRobotVacuumOnLevel(t, "5")

	assert.Equal(t, float64(65), robotFilter.NoiseLevelDB)
	assert.Equal(t, float64(11000), robotFilter.SuctionPowerPA)
	assert.Equal(t, "lidar_ai", robotFilter.NavigationType)
	assert.True(t, robotFilter.RoomMapping)
	assert.True(t, robotFilter.WetCleaning)
	assert.True(t, robotFilter.CarpetDetection)
	assert.True(t, robotFilter.ObstacleAvoidance)
	assert.True(t, robotFilter.AutoEmptyStation)
	assert.True(t, robotFilter.VoiceAssistantSupport)
}
