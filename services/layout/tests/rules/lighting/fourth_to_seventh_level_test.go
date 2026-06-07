package lighting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightingLevel4(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, _, err := placeLightingLevel(apartmentStruct, "4")
	assert.NoError(t, err)

	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r1"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r2"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r3"))
	assert.True(t, layout.HasDeviceInRoom("smart_bulb", "r4"))

	assert.True(t, layout.HasDeviceInRoom("illumination_sensor", "r1"))
	assert.True(t, layout.HasDeviceInRoom("illumination_sensor", "r2"))

	assert.True(t, layout.HasDeviceInRoom("motion_sensor", "r3"))
	assert.True(t, layout.HasDeviceInRoom("motion_sensor", "r4"))

	assert.Equal(t, 3, countDeviceType(layout, "curtains"))
}

func TestLightingLevel4PriceCalculation(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, devicesConfig, err := placeLightingLevel(apartmentStruct, "4")
	assert.NoError(t, err)

	priceInfo := calculateExpectedAndActualPrice(layout, devicesConfig)
	assert.Equal(t, priceInfo.expectedMin, priceInfo.actualMin)
	assert.Equal(t, priceInfo.expectedMax, priceInfo.actualMax)
}
