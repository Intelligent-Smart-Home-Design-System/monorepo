package common

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"github.com/stretchr/testify/require"
)

// Проверяем, что если в комнате два датчика движения
// (motion_sensor и motion_sensor_2), цена считается за оба устройства.
func TestCalculateLayoutPrice_DuplicateDeviceTypeKeys(t *testing.T) {
	devicesConfig, err := configs.LoadDevicesConfig("../../internal/configs/devices.json")
	require.NoError(t, err)

	st := storage.NewStorage()
	e := engine.NewEngine(st, &configs.Tracks{}, devicesConfig)

	layout := apartment.NewApartmentResult()
	layout.Placements["room-1"] = []*device.Placement{
		{
			Device:   &device.Device{ID: "d1", Type: "motion_sensor", Track: "lighting"},
			Position: &point.Point{X: 1, Y: 1},
			Filters:  nil,
		},
		{
			Device:   &device.Device{ID: "d2", Type: "motion_sensor", Track: "lighting"},
			Position: &point.Point{X: 2, Y: 2},
			Filters:  nil,
		},
	}

	priceInfo := e.CalculateLayoutPrice(layout)

	require.Equal(t, devicesConfig.Devices["motion_sensor"].Price.Min*2, priceInfo.MinPrice)
	require.Equal(t, devicesConfig.Devices["motion_sensor"].Price.Max*2, priceInfo.MaxPrice)
}
