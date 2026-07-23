package household

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHouseholdLevel4(t *testing.T) {
	_, robotFilter := placeRobotVacuumOnLevel(t, "4")

	assert.Equal(t, float64(68), robotFilter.NoiseLevelDB)
	assert.Equal(t, 7000, robotFilter.SuctionPowerPA)
	assert.Equal(t, "lidar", robotFilter.NavigationType)
	assert.True(t, robotFilter.RoomMapping)
	assert.True(t, robotFilter.WetCleaning)
	assert.True(t, robotFilter.CarpetDetection)
	assert.True(t, robotFilter.ObstacleAvoidance)
	assert.True(t, robotFilter.AutoEmptyStation)
	assert.False(t, robotFilter.VoiceAssistantSupport)
}
