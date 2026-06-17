package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type AirPurifierRule struct{}

func NewAirPurifierRule() *AirPurifierRule {
	return &AirPurifierRule{}
}

func (r *AirPurifierRule) Type() string {
	return "air_purifier"
}

func (r *AirPurifierRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; ok {
			zr.PollutionZones = collectPollutionZones(zonedAp.OrigAp, zr.OrigRoom)
		}
	}

	return nil
}

func (r *AirPurifierRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := r.Type()

	err := r.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.AirPurifierFilter{}
	}

	airPurifierFilters, ok := configFilters.(*filters.AirPurifierFilter)
	if !ok {
		airPurifierFilters = &filters.AirPurifierFilter{}
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

		position := findAirPurifierPoint(zr)
		if position == nil {
			continue
		}

		layout.AddDeviceToLayout(deviceType, track, zr.OrigRoom.ID, position, nil, airPurifierFilters)
		deviceCnt++
	}

	return nil
}

func collectPollutionZones(ap *apartment.Apartment, room *apartment.Room) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	for _, windowID := range room.Windows {
		window, err := ap.GetWindowByID(windowID)
		if err != nil || len(window.Points) < 2 {
			continue
		}

		zones = append(zones, room.CreateObjectZone(window.Points, window.Width))
	}

	for _, doorID := range room.Doors {
		door, err := ap.GetDoorByID(doorID)
		if err != nil || len(door.Points) < 2 {
			continue
		}

		zones = append(zones, room.CreateObjectZone(door.Points, door.Width))
	}

	return zones
}

func findAirPurifierPoint(zr *apartment.ZonedRoom) *point.Point {
	if zr == nil || zr.OrigRoom == nil || len(zr.OrigRoom.Area) < 3 {
		return nil
	}
	roomCenter := point.GetCenter(zr.OrigRoom.Area)
	if roomCenter == nil {
		return nil
	}

	for _, zone := range zr.PollutionZones {
		zoneCenter := point.GetCenter(zone.Points)
		if zoneCenter == nil {
			continue
		}

		directionToRoom := point.GetDirectionToPoint(*zoneCenter, *roomCenter)
		position := point.MovePointInDirection(*zoneCenter, directionToRoom, climateDeviceOffset)

		if zr.OrigRoom.IsPointInRoom(position) {
			return &position
		}
	}

	return roomCenter
}
