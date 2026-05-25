package security

import (
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)
const (
	degreePerDirection int = 30
    borderShift float64 = 0.1
	defaultRange = 10
	defaultAngle = 120
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
			Range: defaultRange,
			Angle: defaultAngle,
		}
	}
	motionSensorFilters := configFilters.(*filters.MotionSensorFilter)

	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok {
			continue
		}

		bestPoint := findBestMotionPoint(zonedAp.OrigAp, zr, motionSensorFilters)

		if deviceCnt < maxCount {
			layout.AddDeviceToLayout(deviceType, ms.track, zr.OrigRoom.ID, &bestPoint, motionSensorFilters)
			deviceCnt++
		}
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
		doorCenter := point.GetObjectCenter(door.Points)

		zoneCandidates := makeZoneCandidates(doorCenter, PointShift)
		zonePoints := getZonePointsFromCandidates(room, zoneCandidates, doorCenter, door.Width)
		
		zones = append(zones, apartment.NewZone(zonePoints))
	}

	roomCenter := point.GetCenter(room.Area)
	points := []point.Point{
		{X: roomCenter.X - PointShift, Y: roomCenter.Y - PointShift},
		{X: roomCenter.X - PointShift, Y: roomCenter.Y + PointShift},
		{X: roomCenter.X + PointShift, Y: roomCenter.Y + PointShift},
		{X: roomCenter.X + PointShift, Y: roomCenter.Y + PointShift},
	}

	zones = append(zones, apartment.NewZone(points))
	return zones
}

func makeZoneCandidates(doorCenter point.Point, shift float64) []point.Point {
	return []point.Point{
		{X: doorCenter.X, Y: doorCenter.Y + shift},
		{X: doorCenter.X, Y: doorCenter.Y - shift},
		{X: doorCenter.X + shift, Y: doorCenter.Y},
		{X: doorCenter.X - shift, Y: doorCenter.Y},
	}
}

func getZonePointsFromCandidates(room *apartment.Room, zoneCandidates []point.Point, object point.Point, objectWidth float64) []point.Point {
	var points []point.Point
	var p point.Point
	
	for _, candidate := range zoneCandidates {	// цикл из 4 итераций
		if room.IsPointInRoom(candidate) {
			p = candidate
			break
		}
	}

	if p.X < object.X {
		shiftX := -PointShift
		shiftY := objectWidth / 2

		points = []point.Point{
			{X: object.X + borderShift, Y: object.Y - shiftY},
			{X: object.X + shiftX, Y: object.Y - shiftY},
			{X: object.X + shiftX, Y: object.Y + shiftY},
			{X: object.X + borderShift, Y: object.Y + shiftY},
		}
		return points
	}

	if p.X > object.X {
		shiftX := PointShift
		shiftY := objectWidth / 2

		points = []point.Point{
			{X: object.X - borderShift, Y: object.Y - shiftY},
			{X: object.X + shiftX, Y: object.Y - shiftY},
			{X: object.X + shiftX, Y: object.Y + shiftY},
			{X: object.X - borderShift, Y: object.Y + shiftY},
		}
		return points
	} 
	
	if p.Y < object.Y {
		shiftX := objectWidth / 2
		shiftY := -PointShift

		points = []point.Point{
			{X: object.X - shiftX, Y: object.Y - borderShift},
			{X: object.X - shiftX, Y: object.Y + shiftY},
			{X: object.X + shiftX, Y: object.Y + shiftY},
			{X: object.X + shiftX, Y: object.Y - borderShift},
		}
		return points
	}

	shiftX := objectWidth / 2
	shiftY := PointShift

	points = []point.Point{
		{X: object.X - shiftX, Y: object.Y + borderShift},
		{X: object.X - shiftX, Y: object.Y + shiftY},
		{X: object.X + shiftX, Y: object.Y + shiftY},
		{X: object.X + shiftX, Y: object.Y + borderShift},
	}
	return points
}

func findBestMotionPoint (ap *apartment.Apartment, zr *apartment.ZonedRoom, filter *filters.MotionSensorFilter) point.Point {
	r := zr.OrigRoom
	var bestPoint point.Point
	maxCoverage := 0.0
	
	for _, wID := range zr.OrigRoom.Walls {
		if zr.ACAvailableWalls != nil {
			if _, ok := zr.ACAvailableWalls[wID]; !ok {
				continue
			}
		}

		wall, err := ap.GetWallByID(wID)
		if err != nil {
			continue
		}

		center := point.GetObjectCenter(wall.Points)
		coveredPoints := 0.0

		for _, zone := range zr.HighTrafficZones {
			coveredPoints += getZonePointsCover(ap, r, center, zone.Points, degreePerDirection, filter.Range, filter.Angle)
		}
		
		if coveredPoints > maxCoverage {
			maxCoverage = coveredPoints
			bestPoint = center
		}
	}

	for _, corner := range zr.OrigRoom.Area {
		coveredPoints := 0.0

		for _, zone := range zr.HighTrafficZones {
			coveredPoints += getZonePointsCover(ap, r, corner, zone.Points, degreePerDirection, filter.Range, filter.Angle)
		}
		
		if coveredPoints > maxCoverage {
			maxCoverage = coveredPoints
			bestPoint = corner
		}
	}

	return bestPoint
}

func getZonePointsCover(ap *apartment.Apartment, room *apartment.Room, devicePoint point.Point, zonePoints []point.Point, degreePerDirection int, deviceRange float64, deviceAngle float64) float64 {
	directionCnt := 360 / degreePerDirection
	bestCoverage := 0.0

	for i := 0; i < directionCnt; i++ {
		angle := 2 * math.Pi * float64(i) / float64(directionCnt)
		direction := point.Point{
			X: math.Cos(angle),
			Y: math.Sin(angle),
		}

		coverage := getZonePointsCoverWithDirection(ap, room, devicePoint, zonePoints, deviceRange, deviceAngle, direction)

		if coverage > bestCoverage {
			bestCoverage = coverage
		}
	}

	return bestCoverage
}

func getZonePointsCoverWithDirection(ap *apartment.Apartment, room *apartment.Room, devicePoint point.Point, zonePoints []point.Point, deviceRange, deviceAngle float64, deviceDirection point.Point) float64 {
	result := 0
	for _, p := range zonePoints {
		if room.IsPointVisibleOnDevice(ap, p, devicePoint, deviceRange, deviceAngle, deviceDirection) {
			result += 1
		}
	}

	return float64(result) / float64(len(zonePoints))
}
