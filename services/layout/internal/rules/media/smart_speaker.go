package media

import (
	"math"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const speakerShiftFromTV = 0.5

type SmartSpeakerRule struct {
	track string
}

func NewSmartSpeakerRule() *SmartSpeakerRule {
	return &SmartSpeakerRule{
		track: "media",
	}
}

func (ss *SmartSpeakerRule) Type() string {
	return "smart_speaker"
}

func (ss *SmartSpeakerRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := ss.Type()

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(ss.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.SmartSpeaker{}
	}
	smartSpeakerFilters := configFilters.(*filters.SmartSpeaker)

	smartTVRooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, r := range smartTVRooms {
		roomsSet[r.Name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok || deviceCnt >= maxCount {
			continue
		}

		bestPoint := findBestSmartSpeakerPoint(zonedAp.OrigAp, zr, layout)
		if bestPoint == nil {
			continue
		}

		layout.AddDeviceToLayout(deviceType, ss.track, zr.OrigRoom.ID, bestPoint, nil, smartSpeakerFilters)
		deviceCnt++
	}

	return nil
}

func findBestSmartSpeakerPoint(ap *apartment.Apartment, zr *apartment.ZonedRoom, layout *apartment.Layout) *point.Point {
	room := zr.OrigRoom

	tvPosition, direction, width := findTVPosition(ap, room, layout)
	if tvPosition != nil {
		return getPositionNearTV(ap, room, tvPosition, direction, width)
	}

	window, windowPosition := findWindowPosition(ap, room)
	if window != nil {
		return windowPosition
	}

	for _, corner := range room.Area {
		if !isBlockedPoint(ap, room, corner) {
			return &corner
		}
	}

	return point.GetCenter(room.Area)
}

func findTVPosition(ap *apartment.Apartment, room *apartment.Room, layout *apartment.Layout) (*point.Point, *point.Point, float64) {
	for _, fID := range room.Furniture {
		furniture, err := ap.GetFurnitureByID(fID)
		if err != nil {
			continue
		}

		if strings.ToLower(furniture.Name) == apartment.TV {
			tvCenter := point.GetObjectCenter(furniture.Points)
			distance := point.CalculatePointsDistance(furniture.Points[0], furniture.Points[1])

			seg := point.Segment{From: furniture.Points[0], To: furniture.Points[1]}
			direction := seg.Direction()

			return &tvCenter, &direction, distance
		}
	}

	for _, placement := range layout.Placements[room.ID] {
		if strings.ToLower(placement.Device.Type) == apartment.SmartTV {
			tvFilter := placement.Filters.(*filters.SmartTVFilter)
			return placement.Position, placement.Direction, tvFilter.Width
		}
	}

	return nil, nil, 0
}

func getPositionNearTV(ap *apartment.Apartment, room *apartment.Room, tvPosition *point.Point, direction *point.Point, width float64) *point.Point {
	totalShift := (width / 2) + speakerShiftFromTV
	nearestWall := getNearestWall(ap, room, tvPosition)

	if nearestWall != nil {
		wallCenter := point.GetObjectCenter(nearestWall.Points)
		if tvPosition.X < wallCenter.X {
			res := point.MovePointInDirection(*tvPosition, *direction, totalShift)
			if !isBlockedPoint(ap, room, res) {
				return &res
			}
		}

		reverseDirection := point.Point{
			X: -direction.X,
			Y: -direction.Y,
		}
		res := point.MovePointInDirection(*tvPosition, reverseDirection, totalShift)
		if !isBlockedPoint(ap, room, res) {
			return &res
		}

		return &res
	}

	res := point.MovePointInDirection(*tvPosition, *direction, totalShift)
	return &res
}

func getNearestWall(ap *apartment.Apartment, room *apartment.Room, tvPosition *point.Point) *apartment.Wall {
	var result *apartment.Wall
	minDist := math.MaxFloat64

	for _, wID := range room.Walls {
		wall, err := ap.GetWallByID(wID)
		if err != nil {
			continue
		}

		wallSegment := point.Segment{From: wall.Points[0], To: wall.Points[1]}
		closestPoint := point.ClosestPointOnSegment(*tvPosition, wallSegment)
		distance := point.CalculatePointsDistance(closestPoint, *tvPosition)

		if distance < minDist {
			minDist = distance
			result = wall
		}
	}

	return result
}

func findWindowPosition(ap *apartment.Apartment, room *apartment.Room) (*apartment.Window, *point.Point) {
	var result *apartment.Window
	maxWidth := 0.0

	for _, wID := range room.Windows {
		window, err := ap.GetWindowByID(wID)
		if err != nil {
			continue
		}

		if window.Width > maxWidth {
			maxWidth = window.Width
			result = window
		}
	}

	if result != nil {
		return result, &result.Points[0]
	}

	return nil, nil
}

func isBlockedPoint(ap *apartment.Apartment, room *apartment.Room, p point.Point) bool {
	for _, fID := range room.Furniture {
		furniture, err := ap.GetFurnitureByID(fID)
		if err != nil {
			continue
		}

		if point.IsPointInPolygon(p, furniture.Points) {
			return true
		}
	}

	return false
}
