<<<<<<< HEAD
package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type WindowSensorRule struct {
	track string
}

func NewWindowSensorRule() *WindowSensorRule {
	return &WindowSensorRule{
		track: "security",
	}
}

func (ws *WindowSensorRule) Type() string {
	return "window_sensor"
}

func (ws *WindowSensorRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
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
			zr.WindowZones, err = collectWindowZones(zonedAp.OrigAp, zr.OrigRoom)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (ws *WindowSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := ws.Type()
	
	err := ws.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(ws.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.WindowSensorFilter{}
	}
	windowSensorFilters := configFilters.(*filters.WindowSensorFilter)

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		for _, windowZone := range zr.WindowZones {
			if windowZone == nil {
				continue
			}
			zoneCenter := point.GetCenter(windowZone.Points)

			if deviceCnt < maxCount {
				layout.AddDeviceToLayout(deviceType, ws.track, zr.OrigRoom.ID, zoneCenter, nil, windowSensorFilters)
				deviceCnt++
			}
		}
	}

	return nil
}

func collectWindowZones(ap *apartment.Apartment, room *apartment.Room) ([]*apartment.Zone, error) {
	zones := make([]*apartment.Zone, 0)


	for _, wID := range room.Windows {
		window, err := ap.GetWindowByID(wID)
		if err != nil {
			return nil, err
		}

		zones = append(zones, apartment.NewZone(window.Points))
	}

	return zones, nil
}
=======
package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type WindowSensorRule struct {
	track string
}

func NewWindowSensorRule() *WindowSensorRule {
	return &WindowSensorRule{
		track: "security",
	}
}

func (ws *WindowSensorRule) Type() string {
	return "window_sensor"
}

func (ws *WindowSensorRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
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
			zr.WindowZones, err = collectWindowZones(zonedAp.OrigAp, zr.OrigRoom)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (ws *WindowSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := ws.Type()
	
	err := ws.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(ws.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.WindowSensorFilter{}
	}
	windowSensorFilters := configFilters.(*filters.WindowSensorFilter)

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		for _, windowZone := range zr.WindowZones {
			if windowZone == nil {
				continue
			}
			zoneCenter := point.GetCenter(windowZone.Points)

			if deviceCnt < maxCount {
				layout.AddDeviceToLayout(deviceType, ws.track, zr.OrigRoom.ID, zoneCenter, windowSensorFilters)
				deviceCnt++
			}
		}
	}

	return nil
}

func collectWindowZones(ap *apartment.Apartment, room *apartment.Room) ([]*apartment.Zone, error) {
	zones := make([]*apartment.Zone, 0)


	for _, wID := range room.Windows {
		window, err := ap.GetWindowByID(wID)
		if err != nil {
			return nil, err
		}

		zones = append(zones, apartment.NewZone(window.Points))
	}

	return zones, nil
}
>>>>>>> 4bf54f8 (hz)
