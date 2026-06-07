package lighting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightingLevel6(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, _, err := placeLightingLevel(apartmentStruct, "6")
	assert.NoError(t, err)

	assert.Equal(t, 4, countDeviceType(layout, "presence_sensor"))
	assert.True(t, layout.HasDeviceInRoom("presence_sensor", "r3"))
	assert.True(t, layout.HasDeviceInRoom("presence_sensor", "r4"))
	assert.True(t, layout.HasDeviceInRoom("presence_sensor", "r6"))

	assert.True(t, layout.HasDeviceInRoom("built_in_backlight", "r3"))
	assert.True(t, layout.HasDeviceInRoom("built_in_backlight", "r4"))
}

func TestLightingLevel6PriceCalculation(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, devicesConfig, err := placeLightingLevel(apartmentStruct, "6")
	assert.NoError(t, err)

	priceInfo := calculateExpectedAndActualPrice(layout, devicesConfig)
	assert.Equal(t, priceInfo.expectedMin, priceInfo.actualMin)
	assert.Equal(t, priceInfo.expectedMax, priceInfo.actualMax)
}
