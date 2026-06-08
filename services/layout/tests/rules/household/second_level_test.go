package household

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHouseholdLevel2(t *testing.T) {
	_, robotFilter := placeRobotVacuumOnLevel(t, "2")

	assert.Equal(t, float64(72), robotFilter.NoiseLevelDB)
	assert.Equal(t, 3500, robotFilter.SuctionPowerPA)
	assert.Equal(t, "lidar", robotFilter.NavigationType)
	assert.True(t, robotFilter.RoomMapping)
	assert.False(t, robotFilter.WetCleaning)
	assert.False(t, robotFilter.CarpetDetection)
	assert.False(t, robotFilter.ObstacleAvoidance)
	assert.False(t, robotFilter.AutoEmptyStation)
	assert.False(t, robotFilter.VoiceAssistantSupport)
}