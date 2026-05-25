package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type GasLeakSensorRule struct {
	track string
}

func NewGasLeakRule() *GasLeakSensorRule {
	return &GasLeakSensorRule{
		track: "security",
	}
}

func (gl *GasLeakSensorRule) Type() string {
	return "gas_leak_sensor"
}

func (gl *GasLeakSensorRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	rooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, r := range rooms {
		roomsSet[r.Name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; ok {
			zr.GasZones = collectGasZones(zr.GetAppliances())
		}
	}

	return nil
}

func (gl *GasLeakSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := gl.Type()
	
	err := gl.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(gl.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.GasLeakSensorFilter{}
	}
	gasLeakSensorFilters := configFilters.(*filters.GasLeakSensorFilter)

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		for _, gasZone := range zr.GasZones {
			zoneCenter := point.GetCenter(gasZone.Points)

			if deviceCnt < maxCount {
				layout.AddDeviceToLayout(deviceType, gl.track, zr.OrigRoom.ID, zoneCenter, gasLeakSensorFilters)
				deviceCnt++
			}
		}
	}

	return nil
}

func collectGasZones(appliances []*apartment.Appliances) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	for _, a := range appliances {
		switch a.Name {
		case apartment.Stove:
			zones = append(zones, apartment.NewZone(a.Points))
		}
	}

	return zones
}
