package lighting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightingLevel7(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, _, err := placeLightingLevel(apartmentStruct, "7")
	assert.NoError(t, err)

	assert.True(t, layout.HasDeviceInRoom("decorative_luminaire", "r1"))
	assert.True(t, layout.HasDeviceInRoom("decorative_luminaire", "r5"))

	assert.True(t, layout.HasDeviceInRoom("built_in_backlight", "r1"))
	assert.True(t, layout.HasDeviceInRoom("built_in_backlight", "r2"))
	assert.True(t, layout.HasDeviceInRoom("built_in_backlight", "r6"))
}

func TestLightingLevel7PriceCalculation(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, devicesConfig, err := placeLightingLevel(apartmentStruct, "7")
	assert.NoError(t, err)

	priceInfo := calculateExpectedAndActualPrice(layout, devicesConfig)
	assert.Equal(t, priceInfo.expectedMin, priceInfo.actualMin)
	assert.Equal(t, priceInfo.expectedMax, priceInfo.actualMax)
}
