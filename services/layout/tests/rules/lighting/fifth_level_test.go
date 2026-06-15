package lighting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightingLevel5(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, _, err := placeLightingLevel(apartmentStruct, "5")
	assert.NoError(t, err)

	assert.True(t, layout.HasDeviceInRoom("wireless_button_switch", "r1"))
	assert.True(t, layout.HasDeviceInRoom("wireless_button_switch", "r2"))
	assert.True(t, layout.HasDeviceInRoom("wireless_button_switch", "r5"))

	assert.True(t, layout.HasDeviceInRoom("smart_dimmer", "r1"))
	assert.True(t, layout.HasDeviceInRoom("smart_dimmer", "r5"))
}

func TestLightingLevel5PriceCalculation(t *testing.T) {
	apartmentStruct := buildLightingApartmentForHighLevels()
	layout, devicesConfig, err := placeLightingLevel(apartmentStruct, "5")
	assert.NoError(t, err)

	priceInfo := calculateExpectedAndActualPrice(layout, devicesConfig)
	assert.Equal(t, priceInfo.expectedMin, priceInfo.actualMin)
	assert.Equal(t, priceInfo.expectedMax, priceInfo.actualMax)
}
