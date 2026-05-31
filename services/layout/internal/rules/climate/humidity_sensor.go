package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type HumiditySensorRule struct{}

func NewHumiditySensorRule() *HumiditySensorRule {
	return &HumiditySensorRule{}
}

func (r *HumiditySensorRule) Type() string {
	return "humidity_sensor"
}

func (r *HumiditySensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := r.Type()

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.HumiditySensorFilter{}
	}

	humiditySensorFilters, ok := configFilters.(*filters.HumiditySensorFilter)
	if !ok {
		humiditySensorFilters = &filters.HumiditySensorFilter{}
	}

	rooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, room := range rooms {
		roomsSet[room.ID] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if deviceCnt >= maxCount {
			return nil
		}

		if _, ok := roomsSet[zr.OrigRoom.ID]; !ok {
			continue
		}

		wall := findFirstWall(zr)
		if wall == nil {
			continue
		}

		position := point.GetObjectCenter(wall.Points)
		layout.AddDeviceToLayout(deviceType, track, zr.OrigRoom.ID, &position, humiditySensorFilters)
		deviceCnt++
	}

	return nil
}
