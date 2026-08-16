package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type CO2SensorRule struct{}

func NewCO2SensorRule() *CO2SensorRule {
	return &CO2SensorRule{}
}

func (r *CO2SensorRule) Type() string {
	return "co2_sensor"
}

func (r *CO2SensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := r.Type()

	tracksConfig := configs.GetGlobalTracksConfig()
	var co2SensorFilters *filters.CO2SensorFilter

	if levelNum != "" {
		configFilters, err := tracksConfig.GetDeviceFilter(track, levelNum, deviceType)
		if err == nil && configFilters != nil {
			typedFilters, ok := configFilters.(*filters.CO2SensorFilter)
			if ok {
				co2SensorFilters = typedFilters
			}
		}
	}

	if co2SensorFilters == nil {
		co2SensorFilters = &filters.CO2SensorFilter{}
	}

	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if deviceCnt >= maxCount {
			return nil
		}

		if _, ok := roomsSet[zr.OrigRoom.Type]; !ok {
			continue
		}

		wall := findFirstWall(zr)
		if wall == nil {
			continue
		}

		position := point.GetObjectCenter(wall.Points)
		layout.AddDeviceToLayout(deviceType, track, zr.OrigRoom.ID, &position, nil, co2SensorFilters)
		deviceCnt++
	}

	return nil
}
