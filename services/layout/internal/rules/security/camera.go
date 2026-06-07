<<<<<<< HEAD
package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const (
	defaultCameraAngle  = 100
	defaultCameraRange  = 8
	minRequiredCoverage = 0.3
)

type CameraRule struct {
	track string
}

func NewCameraRule() *CameraRule {
	return &CameraRule{
		track: "security",
	}
}

func (c *CameraRule) Type() string {
	return "camera"
}

func (c *CameraRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
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
			zr.ViewedZones = collectViewedZones(zonedAp.OrigAp, zr.OrigRoom)

		}
	}

	return nil
}

func (c *CameraRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := c.Type()

	err := c.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(c.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.CameraFilter{
			Angle: defaultCameraAngle,
			Range: defaultCameraRange,
		}
	}
	cameraFilters := configFilters.(*filters.CameraFilter)

	cameraRooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, r := range cameraRooms {
		roomsSet[r.Name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok || deviceCnt >= maxCount {
			continue
		}

		// При необходимости можно вовзращать направление камеры, чтобы другим модулям легче было взаимодейстовать
		bestPoint, direction, distance := findBestCameraPoint(zonedAp.OrigAp, zr, cameraFilters)
		if bestPoint == nil {
			continue
		}

		var deviceFilter *filters.CameraFilter
		if distance != -1 {
			deviceFilter = &filters.CameraFilter{
				Angle: cameraFilters.Angle,
				Range: cameraFilters.Range,
				NightVision: cameraFilters.NightVision,
				Resolution: cameraFilters.Resolution,
				RecommendedRangeM: distance,
				Direction: &direction,
			}
		} else {
			deviceFilter = &filters.CameraFilter{
				Angle: cameraFilters.Angle,
				Range: cameraFilters.Range,
				NightVision: cameraFilters.NightVision,
				Resolution: cameraFilters.Resolution,
				RecommendedRangeM: cameraFilters.Range,
				Direction: &direction,
			}
		}

		layout.AddDeviceToLayout(deviceType, c.track, zr.OrigRoom.ID, bestPoint, &direction, deviceFilter)
		deviceCnt++
	}

	return nil
}

func collectViewedZones(ap *apartment.Apartment, room *apartment.Room) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	if room.Name == apartment.RoomHall {
		entryDoor := room.GetEntryDoor(ap)
		if entryDoor != nil {
			zones = append(zones, room.CreateObjectZone(entryDoor.Points, entryDoor.Width))
		}
	}

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

	roomCenter := room.Center
	if roomCenter == nil {
		roomCenter = point.GetCenter(room.Area)
	}

	zones = append(zones, apartment.NewZone(point.PointToSquare(*roomCenter, Meter)))
	return zones
}

func findBestCameraPoint(ap *apartment.Apartment, zr *apartment.ZonedRoom, filter *filters.CameraFilter) (*point.Point, point.Point, float64) {
	room := zr.OrigRoom

	if room.Name == apartment.RoomHall {
		entryDoor := room.GetEntryDoor(ap)
		if entryDoor != nil {
			doorCenter := point.GetObjectCenter(entryDoor.Points)
			bestPoint, distance := room.GetTheOppositePoint(doorCenter)

			direction := point.GetDirectionToPoint(bestPoint, doorCenter)
			filter.RecommendedRangeM = distance

			return &bestPoint, direction, distance
		}
	}

	var bestPoint, bestDirection point.Point
	maxCoverage := 0.0

	for _, corner := range room.Area {
		direction, coverage := apartment.FindBestDirectionForDevicePoint(ap, zr, zr.ViewedZones, corner, filter.Range, filter.Angle)

		if maxCoverage < coverage {
			maxCoverage = coverage
			bestPoint = corner
			bestDirection = direction
		}
	}

	if maxCoverage < minRequiredCoverage {
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
			direction, coverage := apartment.FindBestDirectionForDevicePoint(ap, zr, zr.ViewedZones, wallCenter, filter.Range, filter.Angle)

			if maxCoverage < coverage {
				maxCoverage = coverage
				bestPoint = wallCenter
				bestDirection = direction
			}
		}
	}

	return &bestPoint, bestDirection, -1
}
=======
package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const (
	defaultCameraAngle  = 100
	defaultCameraRange  = 8
	minRequiredCoverage = 0.3
)

type CameraRule struct {
	track string
}

func NewCameraRule() *CameraRule {
	return &CameraRule{
		track: "security",
	}
}

func (c *CameraRule) Type() string {
	return "camera"
}

func (c *CameraRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
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
			zr.ViewedZones = collectViewedZones(zonedAp.OrigAp, zr.OrigRoom)

		}
	}

	return nil
}

func (c *CameraRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := c.Type()

	err := c.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(c.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.CameraFilter{
			Angle: defaultCameraAngle,
			Range: defaultCameraRange,
		}
	}
	cameraFilters := configFilters.(*filters.CameraFilter)

	cameraRooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, r := range cameraRooms {
		roomsSet[r.Name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok || deviceCnt >= maxCount {
			continue
		}

		// При необходимости можно вовзращать направление камеры, чтобы другим модулям легче было взаимодейстовать
		bestPoint, _ := findBestCameraPoint(zonedAp.OrigAp, zr, cameraFilters)
		if bestPoint == nil {
			continue
		}

		layout.AddDeviceToLayout(deviceType, c.track, zr.OrigRoom.ID, bestPoint, cameraFilters)
		deviceCnt++
	}

	return nil
}

func collectViewedZones(ap *apartment.Apartment, room *apartment.Room) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)

	if room.Name == apartment.RoomHall {
		entryDoor := room.GetEntryDoor(ap)
		if entryDoor != nil {
			zones = append(zones, room.CreateObjectZone(entryDoor.Points, entryDoor.Width))
		}
	}

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

	roomCenter := room.Center
	if roomCenter == nil {
		roomCenter = point.GetCenter(room.Area)
	}

	zones = append(zones, apartment.NewZone(point.PointToSquare(*roomCenter, Meter)))
	return zones
}

func findBestCameraPoint(ap *apartment.Apartment, zr *apartment.ZonedRoom, filter *filters.CameraFilter) (*point.Point, point.Point) {
	room := zr.OrigRoom

	if room.Name == apartment.RoomHall {
		entryDoor := room.GetEntryDoor(ap)
		if entryDoor != nil {
			doorCenter := point.GetObjectCenter(entryDoor.Points)
			bestPoint, Distance := room.GetTheOppositePoint(doorCenter)
			fmt.Println(bestPoint)

			direction := point.GetDirectionToPoint(bestPoint, doorCenter)
			filter.RecommendedRange = Distance

			return &bestPoint, direction
		}
	}

	var bestPoint, bestDirection point.Point
	maxCoverage := 0.0

	for _, corner := range room.Area {
		direction, coverage := apartment.FindBestDirectionForDevicePoint(ap, zr, zr.ViewedZones, corner, filter.Range, filter.Angle)

		if maxCoverage < coverage {
			maxCoverage = coverage
			bestPoint = corner
			bestDirection = direction
		}
	}

	if maxCoverage < minRequiredCoverage {
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
			direction, coverage := apartment.FindBestDirectionForDevicePoint(ap, zr, zr.ViewedZones, wallCenter, filter.Range, filter.Angle)

			if maxCoverage < coverage {
				maxCoverage = coverage
				bestPoint = wallCenter
				bestDirection = direction
			}
		}
	}

	return &bestPoint, bestDirection
}
>>>>>>> 4bf54f8 (hz)
