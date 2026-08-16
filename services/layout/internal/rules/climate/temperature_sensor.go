package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type TemperatureSensorRule struct{}

func NewTemperatureSensorRule() *TemperatureSensorRule {
	return &TemperatureSensorRule{}
}

func (r *TemperatureSensorRule) Type() string {
	return "temperature_sensor"
}

func (r *TemperatureSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := r.Type()

	tracksConfig := configs.GetGlobalTracksConfig()
	var temperatureSensorFilters *filters.TemperatureSensorFilter

	if levelNum != "" {
		configFilters, err := tracksConfig.GetDeviceFilter(track, levelNum, deviceType)
		if err == nil && configFilters != nil {
			typedFilters, ok := configFilters.(*filters.TemperatureSensorFilter)
			if ok {
				temperatureSensorFilters = typedFilters
			}
		}
	}

	if temperatureSensorFilters == nil {
		temperatureSensorFilters = &filters.TemperatureSensorFilter{}
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
		layout.AddDeviceToLayout(deviceType, track, zr.OrigRoom.ID, &position, nil, temperatureSensorFilters)
		deviceCnt++
	}

	return nil
}
