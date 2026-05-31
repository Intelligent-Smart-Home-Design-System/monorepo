package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type SmartHumidifierRule struct{}

func NewSmartHumidifierRule() *SmartHumidifierRule {
	return &SmartHumidifierRule{}
}

func (r *SmartHumidifierRule) Type() string {
	return "smart_humidifier"
}

func (r *SmartHumidifierRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	rooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, room := range rooms {
		roomsSet[room.ID] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.ID]; ok {
			zr.RestrictedZones = collectClimateRestrictedZones(zr)
		}
	}

	return nil
}

func (r *SmartHumidifierRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
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
		configFilters = &filters.SmartHumidifierFilter{}
	}

	smartHumidifierFilters, ok := configFilters.(*filters.SmartHumidifierFilter)
	if !ok {
		smartHumidifierFilters = &filters.SmartHumidifierFilter{}
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

		position := findSmartHumidifierPoint(zr)
		if position == nil {
			continue
		}

		layout.AddDeviceToLayout(deviceType, track, zr.OrigRoom.ID, position, smartHumidifierFilters)
		deviceCnt++
	}

	return nil
}

func collectClimateRestrictedZones(zr *apartment.ZonedRoom) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	for _, f := range zr.GetFurniture() {
		if len(f.Points) > 0 {
			zones = append(zones, apartment.NewZone(f.Points))
		}
	}

	for _, p := range zr.GetPlumbing() {
		if len(p.Points) > 0 {
			zones = append(zones, apartment.NewZone(p.Points))
		}
	}

	for _, a := range zr.GetAppliances() {
		if len(a.Points) > 0 {
			zones = append(zones, apartment.NewZone(a.Points))
		}
	}

	return zones
}

func findSmartHumidifierPoint(zr *apartment.ZonedRoom) *point.Point {
	if zr == nil || zr.OrigRoom == nil || len(zr.OrigRoom.Area) < 3 {
		return nil
	}

	roomCenter := point.GetCenter(zr.OrigRoom.Area)
	if roomCenter == nil {
		return nil
	}

	for _, corner := range zr.OrigRoom.Area {
		directionToRoom := point.GetDirectionToPoint(corner, *roomCenter)
		candidate := point.MovePointInDirection(corner, directionToRoom, climateDeviceOffset)

		if !zr.OrigRoom.IsPointInRoom(candidate) {
			continue
		}

		if isPointInZones(candidate, zr.RestrictedZones) {
			continue
		}

		return &candidate
	}

	return roomCenter
}

func isPointInZones(p point.Point, zones []*apartment.Zone) bool {
	for _, zone := range zones {
		if zone.ContainsPoint(p) {
			return true
		}
	}

	return false
}
