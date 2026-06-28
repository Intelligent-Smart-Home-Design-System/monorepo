package household

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const robotVacuumWallOffset = 0.4

type RobotVacuumRule struct {
	track string
}

func NewRobotVacuumRule() *RobotVacuumRule {
	return &RobotVacuumRule{
		track: "household",
	}
}

func (r *RobotVacuumRule) Type() string {
	return "robot_vacuum"
}

func (r *RobotVacuumRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.ID]; ok {
			zr.CleaningZones = collectCleaningZones(zr.OrigRoom)
			zr.RestrictedZones = collectRobotRestrictedZones(zr)
		}
	}

	return nil
}

func (r *RobotVacuumRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := r.Type()

	err := r.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(r.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.RobotVacuumFilter{}
	}

	robotVacuumFilters, ok := configFilters.(*filters.RobotVacuumFilter)
	if !ok {
		robotVacuumFilters = &filters.RobotVacuumFilter{}
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

		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok {
			continue
		}

		position := findRobotVacuumPoint(zr)
		if position == nil {
			continue
		}

		layout.AddDeviceToLayout(deviceType, r.track, zr.OrigRoom.ID, position, nil, robotVacuumFilters)
		deviceCnt++
	}

	return nil
}

// collectCleaningZones создаёт зоны уборки для комнаты (вся площадь комнаты)
func collectCleaningZones(room *apartment.Room) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	if room == nil || len(room.Area) == 0 {
		return zones
	}

	zones = append(zones, apartment.NewZone(room.Area))
	return zones
}

// collectRobotRestrictedZones собирает зоны, куда нельзя ставить робота
func collectRobotRestrictedZones(zr *apartment.ZonedRoom) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	for _, f := range zr.GetFurniture() {
		if len(f.Points) > 0 {
			zones = append(zones, apartment.NewZone(f.Points))
		}
	}

	for _, a := range zr.GetFurniture() {
		if len(a.Points) > 0 {
			zones = append(zones, apartment.NewZone(a.Points))
		}
	}

	return zones
}

// findRobotVacuumPoint ищет точку у стены (с отступом 40см в центр комнаты)
func findRobotVacuumPoint(zr *apartment.ZonedRoom) *point.Point {
	if zr == nil || zr.OrigRoom == nil || len(zr.OrigRoom.Area) < 3 {
		return nil
	}
	roomCenter := point.GetCenter(zr.OrigRoom.Area)
	if roomCenter == nil {
		return nil
	}

	for _, wall := range zr.GetWalls() {
		if len(wall.Points) < 2 {
			continue
		}

		wallCenter := point.GetObjectCenter(wall.Points)
		directionToRoom := point.GetDirectionToPoint(wallCenter, *roomCenter)
		candidate := point.MovePointInDirection(wallCenter, directionToRoom, robotVacuumWallOffset)

		if !zr.OrigRoom.IsPointInRoom(candidate) {
			continue
		}

		if isPointInRestrictedZones(candidate, zr.RestrictedZones) {
			continue
		}

		return &candidate
	}

	return nil
}

// isPointInRestrictedZones проверяет, попадает ли точка в одну из запрещённых зон
func isPointInRestrictedZones(p point.Point, zones []*apartment.Zone) bool {
	for _, zone := range zones {
		if zone.ContainsPoint(p) {
			return true
		}
	}

	return false
}
