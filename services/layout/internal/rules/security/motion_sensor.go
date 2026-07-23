package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const (
	defaultRange = 10000
	defaultAngle = 120
	Meter        = 1000
)

type MotionSensorRule struct {
	track string
}

func NewMotionSensorRule() *MotionSensorRule {
	return &MotionSensorRule{
		track: "security",
	}
}

func (ms *MotionSensorRule) Type() string {
	return "motion_sensor"
}

func (ms *MotionSensorRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; ok {
			zr.HighTrafficZones = collectHighTrafficZones(zonedAp.OrigAp, zr.OrigRoom)
		}
	}

	return nil
}

func (ms *MotionSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := ms.Type()

	err := ms.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(ms.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.MotionSensorFilter{
			DetectionRangeMM: defaultRange,
			DetectionAngleDeg: defaultAngle,
		}
	}
	motionSensorFilters := configFilters.(*filters.MotionSensorFilter)

	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok || deviceCnt >= maxCount {
			continue
		}

		bestPoint, direction := findBestMotionPoint(zonedAp.OrigAp, zr, motionSensorFilters)
		if bestPoint == nil {
			continue
		}

		deviceFilter := &filters.MotionSensorFilter{
			DetectionAngleDeg: motionSensorFilters.DetectionAngleDeg,
			DetectionRangeMM: motionSensorFilters.DetectionRangeMM,
		}

		layout.AddDeviceToLayout(deviceType, ms.track, zr.OrigRoom.ID, bestPoint, &direction, deviceFilter)
		deviceCnt++
	}

	return nil
}

func collectHighTrafficZones(ap *apartment.Apartment, room *apartment.Room) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	for _, dID := range room.Doors {
		door, err := ap.GetDoorByID(dID)
		if err != nil {
			continue
		}

		zones = append(zones, room.CreateObjectZone(door.Points, door.Width))
	}

	for _, wID := range room.Windows {
		window, err := ap.GetWindowByID(wID)
		if err != nil {
			continue
		}

		zones = append(zones, room.CreateObjectZone(window.Points, window.Width))
	}

	return zones
}

func findBestMotionPoint(ap *apartment.Apartment, zr *apartment.ZonedRoom, filter *filters.MotionSensorFilter) (*point.Point, point.Point) {
	room := zr.OrigRoom

	var bestPoint, bestDirection point.Point
	maxCoverage := 0.0

	for _, wID := range room.Walls {
		if zr.ACAvailableWalls != nil {
			if _, ok := zr.ACAvailableWalls[wID]; !ok {
				continue
			}
		}

		wall, err := ap.GetWallByID(wID)
		if err != nil {
			continue
		}

		wallCenter := point.GetObjectCenter(wall.Points)
		direction, coverage := apartment.FindBestDirectionForDevicePoint(ap, zr, zr.HighTrafficZones, wallCenter, filter.DetectionRangeMM, filter.DetectionAngleDeg)

		if maxCoverage < coverage {
			maxCoverage = coverage
			bestPoint = wallCenter
			bestDirection = direction
		}
	}

	if maxCoverage < minRequiredCoverage {
		for _, corner := range room.Area {
			direction, coverage := apartment.FindBestDirectionForDevicePoint(ap, zr, zr.HighTrafficZones, corner, filter.DetectionRangeMM, filter.DetectionAngleDeg)

			if maxCoverage < coverage {
				maxCoverage = coverage
				bestPoint = corner
				bestDirection = direction
			}
		}
	}

	return &bestPoint, bestDirection
}
