package media

import (
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const (
	wallOffset      = 0.5
	maxDistanceToTV = 4
	testPointOffset = 1
)

type SubwooferRule struct {
	track string
}

func NewSubwooferRule() *SubwooferRule {
	return &SubwooferRule{
		track: "media",
	}
}

func (sw *SubwooferRule) Type() string {
	return "subwoofer"
}

func (sw *SubwooferRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := sw.Type()

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(sw.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.Subwoofer{}
	}
	subwooferFilters := configFilters.(*filters.Subwoofer)

	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok || deviceCnt >= maxCount {
			continue
		}

		bestCorner := findBestSubwooferPoint(zonedAp.OrigAp, zr.OrigRoom, layout)
		if bestCorner == nil {
			continue
		}

		layout.AddDeviceToLayout(deviceType, sw.track, zr.OrigRoom.ID, bestCorner, nil, subwooferFilters)
		deviceCnt++
	}

	return nil
}

func findBestSubwooferPoint(ap *apartment.Apartment, room *apartment.Room, layout *apartment.Layout) *point.Point {
	tvCenter, _, _ := findTVPosition(ap, room, layout)
	if tvCenter == nil || len(room.Area) < 3 {
		return nil
	}

	var bestCorner *point.Point
	minDistance := math.MaxFloat64

	n := len(room.Area)
	for i := range room.Area {
		prev := room.Area[(i-1+n)%n]
		curr := room.Area[i]
		next := room.Area[(i+1)%n]

		vector1 := point.Normalize(point.Point{X: prev.X - curr.X, Y: prev.Y - curr.Y})
		vector2 := point.Normalize(point.Point{X: next.X - curr.X, Y: next.Y - curr.Y})

		bissVector := point.Normalize(point.Point{X: vector1.X + vector2.X, Y: vector1.Y + vector2.Y})

		p := point.Point{
			X: curr.X + bissVector.X*wallOffset,
			Y: curr.Y + bissVector.Y*wallOffset,
		}
		if !point.IsPointInPolygon(p, room.Area) {
			p = point.Point{
				X: curr.X - bissVector.X*wallOffset,
				Y: curr.Y - bissVector.Y*wallOffset,
			}

			if !point.IsPointInPolygon(p, room.Area) {
				continue
			}
		}

		tvToCornerVector := point.Point{X: p.X - tvCenter.X, Y: p.Y - tvCenter.Y}
		distance := math.Sqrt(tvToCornerVector.X*tvToCornerVector.X + tvToCornerVector.Y*tvToCornerVector.Y)
		if distance == 0 {
			continue
		}

		tvForward := calculateTVDirection(ap, room, tvCenter)
		tvToCornerDirection := point.Point{X: tvToCornerVector.X / distance, Y: tvToCornerVector.Y / distance}
		dotProduct := tvToCornerDirection.DotProduct(*tvForward)

		if dotProduct >= 0 {
			if distance < minDistance {
				minDistance = distance
				bestCorner = &p
			}
		}
	}

	return bestCorner
}

func calculateTVDirection(ap *apartment.Apartment, room *apartment.Room, tvCenter *point.Point) *point.Point {
	wall := getNearestWall(ap, room, tvCenter)
	if wall == nil {
		return nil
	}

	wallVector := point.Normalize(point.Point{
		X: wall.Points[1].X - wall.Points[0].X,
		Y: wall.Points[1].Y - wall.Points[0].Y,
	})

	perp := point.Point{
		X: -wallVector.Y,
		Y: wallVector.X,
	}

	testPoint := point.MovePointInDirection(*tvCenter, perp, testPointOffset)
	if point.IsPointInPolygon(testPoint, room.Area) {
		return &perp
	}

	return &point.Point{X: -perp.X, Y: -perp.Y}
}
