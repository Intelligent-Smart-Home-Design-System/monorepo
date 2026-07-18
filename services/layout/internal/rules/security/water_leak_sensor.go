package security

import (
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const PointShift float64 = 500

type WaterLeakSensorRule struct {
	track string
}

func NewWaterLeakRule() *WaterLeakSensorRule {
	return &WaterLeakSensorRule{
		track: "security",
	}
}

func (wl *WaterLeakSensorRule) Type() string {
	return "water_leak_sensor"
}

func (wl *WaterLeakSensorRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; ok {
			zr.WetZones = collectWetZones(zr.GetFurniture())
		}
	}

	return nil
}

func (wl *WaterLeakSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := wl.Type()

	err := wl.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(wl.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.WaterLeakSensorFilter{}
	}
	waterLeakSensorFilters := configFilters.(*filters.WaterLeakSensorFilter)

	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok {
			continue
		}

		for _, wetZone := range zr.WetZones {
			if deviceCnt >= maxCount {
				return nil
			}

			if len(wetZone.Points) == 0 {
				continue
			}

			zoneCenter := point.GetCenter(wetZone.Points)

			if deviceCnt < maxCount {
				layout.AddDeviceToLayout(deviceType, wl.track, zr.OrigRoom.ID, zoneCenter, nil, waterLeakSensorFilters)
				deviceCnt++
			}
		}
	}

	return nil
}

func collectWetZones(furniture []*apartment.Furniture) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	for _, f := range furniture {
		name := strings.ToLower(f.Category)
		switch name {
		case apartment.Toilet, apartment.Sink, apartment.Bathtub, apartment.Shower:
			zones = append(zones, apartment.NewZone(f.Points))
		}
	}

	return zones
}
