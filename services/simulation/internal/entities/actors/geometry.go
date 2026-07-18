package actors

import (
	"math"
	"sort"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
)

// segment описывает отрезок между двумя точками.
type segment struct {
	x1, y1, x2, y2 float64
}

// splitWallByDoors разбивает стену на части без дверных проемов и возвращает сегменты стены.
func splitWallByDoors(wall *api.Wall, doors []*api.Door) []segment {
	wallStart := wall.Points[0]
	wallEnd := wall.Points[1]

	wallDX := wallEnd[0] - wallStart[0]
	wallDY := wallEnd[1] - wallStart[1]
	wallLenSq := wallDX*wallDX + wallDY*wallDY

	type interval struct {
		t1 float64
		t2 float64
	}

	var gaps []interval

	for _, door := range doors {
		if !doorOnWall(door, wall) {
			continue
		}

		t1 := projectOnSegment(door.Points[0], wallStart, wallEnd, wallLenSq)
		t2 := projectOnSegment(door.Points[1], wallStart, wallEnd, wallLenSq)

		if t1 > t2 {
			t1, t2 = t2, t1
		}

		if t2 > 0 && t1 < 1 {
			gaps = append(gaps, interval{
				math.Max(0, t1),
				math.Min(1, t2),
			})
		}
	}

	if len(gaps) == 0 {
		return []segment{{wallStart[0], wallStart[1], wallEnd[0], wallEnd[1]}}
	}

	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].t1 < gaps[j].t1
	})

	var segments []segment
	prev := 0.0

	for _, gap := range gaps {
		if gap.t1 > prev+1e-10 {
			segments = append(segments, segment{
				wallStart[0] + prev*wallDX,
				wallStart[1] + prev*wallDY,
				wallStart[0] + gap.t1*wallDX,
				wallStart[1] + gap.t1*wallDY,
			})
		}

		if gap.t2 > prev {
			prev = gap.t2
		}
	}

	if prev < 1.0-1e-10 {
		segments = append(segments, segment{
			wallStart[0] + prev*wallDX,
			wallStart[1] + prev*wallDY,
			wallEnd[0],
			wallEnd[1],
		})
	}

	return segments
}

// projectOnSegment проецирует точку p на отрезок и возвращает параметр t в диапазоне [0..1].
func projectOnSegment(p [2]float64, start, end [2]float64, lenSq float64) float64 {
	dx := end[0] - start[0]
	dy := end[1] - start[1]
	t := ((p[0]-start[0])*dx + (p[1]-start[1])*dy) / lenSq

	return math.Max(0, math.Min(1, t))
}

// findRoomByID находит комнату по ID и возвращает указатель на нее или nil.
func findRoomByID(floor *api.Floor, roomID string) *api.Room {
	for i, room := range floor.Rooms {
		if room.ID == roomID {
			return &floor.Rooms[i]
		}
	}

	return nil
}

// findWallByID находит стену по ID и возвращает указатель на нее или nil.
func findWallByID(floor *api.Floor, wallID string) *api.Wall {
	for i, wall := range floor.Walls {
		if wall.ID == wallID {
			return &floor.Walls[i]
		}
	}

	return nil
}

// doorOnWall проверяет, лежит ли дверь на стене, и возвращает результат.
func doorOnWall(door *api.Door, wall *api.Wall) bool {
	wallSeg := segment{
		wall.Points[0][0], wall.Points[0][1],
		wall.Points[1][0], wall.Points[1][1],
	}
	doorSeg := segment{
		door.Points[0][0], door.Points[0][1],
		door.Points[1][0], door.Points[1][1],
	}

	return segmentsCollinearAndOverlap(wallSeg, doorSeg)
}

// intersectSegments находит пересечение отрезков и возвращает параметр t первого отрезка и признак пересечения.
func intersectSegments(a, b segment) (float64, bool) {
	dx1 := a.x2 - a.x1
	dy1 := a.y2 - a.y1
	dx2 := b.x2 - b.x1
	dy2 := b.y2 - b.y1

	denominator := dx1*dy2 - dy1*dx2
	if math.Abs(denominator) < 1e-10 {
		return 0, false
	}

	t := ((b.x1-a.x1)*dy2 - (b.y1-a.y1)*dx2) / denominator
	u := ((b.x1-a.x1)*dy1 - (b.y1-a.y1)*dx1) / denominator

	if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
		return t, true
	}

	return 0, false
}

// segmentsCollinearAndOverlap проверяет коллинеарность и перекрытие отрезков и возвращает результат.
func segmentsCollinearAndOverlap(a, b segment) bool {
	cross := (a.x2-a.x1)*(b.y1-a.y1) - (a.y2-a.y1)*(b.x1-a.x1)
	if math.Abs(cross) > 1e-10 {
		return false
	}

	aMinX, aMaxX := math.Min(a.x1, a.x2), math.Max(a.x1, a.x2)
	bMinX, bMaxX := math.Min(b.x1, b.x2), math.Max(b.x1, b.x2)
	aMinY, aMaxY := math.Min(a.y1, a.y2), math.Max(a.y1, a.y2)
	bMinY, bMaxY := math.Min(b.y1, b.y2), math.Max(b.y1, b.y2)

	return aMinX <= bMaxX && bMinX <= aMaxX &&
		aMinY <= bMaxY && bMinY <= aMaxY
}

// polygonBounds считает bounding box polygon и возвращает minX, maxX, minY, maxY.
func polygonBounds(points [][2]float64) (float64, float64, float64, float64) {
	minX, maxX := points[0][0], points[0][0]
	minY, maxY := points[0][1], points[0][1]
	for _, point := range points[1:] {
		minX = math.Min(minX, point[0])
		maxX = math.Max(maxX, point[0])
		minY = math.Min(minY, point[1])
		maxY = math.Max(maxY, point[1])
	}

	return minX, maxX, minY, maxY
}

// polygonIntersectsSegment проверяет пересечение polygon с отрезком и возвращает результат.
func polygonIntersectsSegment(polygon [][2]float64, wall segment) bool {
	for i := range polygon {
		next := (i + 1) % len(polygon)
		edge := segment{polygon[i][0], polygon[i][1], polygon[next][0], polygon[next][1]}
		if _, intersects := intersectSegments(edge, wall); intersects {
			return true
		}
	}

	return false
}

// clipPolygonByLineKeepingPoint обрезает polygon по линии, оставляя сторону keepPoint, и возвращает новый polygon.
func clipPolygonByLineKeepingPoint(polygon [][2]float64, line segment, keepPoint [2]float64) [][2]float64 {
	if len(polygon) == 0 {
		return polygon
	}

	keepDistance := signedDistanceToLine(keepPoint, line)
	if math.Abs(keepDistance) <= blockEpsilon {
		return polygon
	}

	inside := func(point [2]float64) bool {
		distance := signedDistanceToLine(point, line)
		return distance*keepDistance >= -blockEpsilon
	}

	var clipped [][2]float64
	prev := polygon[len(polygon)-1]
	prevInside := inside(prev)

	for _, current := range polygon {
		currentInside := inside(current)
		if currentInside != prevInside {
			if point, ok := lineIntersection(prev, current, line); ok {
				clipped = append(clipped, point)
			}
		}
		if currentInside {
			clipped = append(clipped, current)
		}
		prev = current
		prevInside = currentInside
	}

	return clipped
}

// signedDistanceToLine считает ориентированное расстояние точки до линии и возвращает его знак/величину.
func signedDistanceToLine(point [2]float64, line segment) float64 {
	return (line.x2-line.x1)*(point[1]-line.y1) - (line.y2-line.y1)*(point[0]-line.x1)
}

// pointToSegmentDistance считает кратчайшее расстояние от точки до отрезка.
func pointToSegmentDistance(point [2]float64, line segment) float64 {
	dx := line.x2 - line.x1
	dy := line.y2 - line.y1
	if math.Abs(dx) < blockEpsilon && math.Abs(dy) < blockEpsilon {
		return math.Hypot(point[0]-line.x1, point[1]-line.y1)
	}

	t := ((point[0]-line.x1)*dx + (point[1]-line.y1)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))

	closestX := line.x1 + t*dx
	closestY := line.y1 + t*dy
	return math.Hypot(point[0]-closestX, point[1]-closestY)
}

// lineIntersection ищет пересечение отрезка ab с линией и возвращает точку и признак успеха.
func lineIntersection(a, b [2]float64, line segment) ([2]float64, bool) {
	edge := segment{a[0], a[1], b[0], b[1]}
	dx1 := edge.x2 - edge.x1
	dy1 := edge.y2 - edge.y1
	dx2 := line.x2 - line.x1
	dy2 := line.y2 - line.y1

	denominator := dx1*dy2 - dy1*dx2
	if math.Abs(denominator) < blockEpsilon {
		return [2]float64{}, false
	}

	t := ((line.x1-edge.x1)*dy2 - (line.y1-edge.y1)*dx2) / denominator
	return [2]float64{
		edge.x1 + t*dx1,
		edge.y1 + t*dy1,
	}, true
}
