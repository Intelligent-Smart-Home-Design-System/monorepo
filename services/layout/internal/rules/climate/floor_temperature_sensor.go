package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type FloorTemperatureSensorRule struct{}

func NewFloorTemperatureSensorRule() *FloorTemperatureSensorRule {
	return &FloorTemperatureSensorRule{}
}

func (r *FloorTemperatureSensorRule) Type() string {
	return "floor_temperature_sensor"
}

func (r *FloorTemperatureSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := r.Type()

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.FloorTemperatureSensorFilter{}
	}

	floorTemperatureSensorFilters, ok := configFilters.(*filters.FloorTemperatureSensorFilter)
	if !ok {
		floorTemperatureSensorFilters = &filters.FloorTemperatureSensorFilter{}
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

		if zr.OrigRoom == nil || len(zr.OrigRoom.Area) < 3 {
			continue
		}

		position := point.GetCenter(zr.OrigRoom.Area)
		if position == nil {
			continue
		}

		layout.AddDeviceToLayout(deviceType, track, zr.OrigRoom.ID, position, nil, floorTemperatureSensorFilters)
		deviceCnt++
	}

	return nil
}
