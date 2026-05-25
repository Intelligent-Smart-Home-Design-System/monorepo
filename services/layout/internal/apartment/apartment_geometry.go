package apartment

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

// GetBoundaries возвращает противоположные точки прямоугольника, описывающего комнату
func (r *Room) GetBoundaries() (point.Point, point.Point) {
	minX, minY, maxX, maxY := r.Area[0].X, r.Area[0].Y, r.Area[0].X, r.Area[0].Y
	for _, p := range r.Area[1:] {
		minX = min(minX, p.X)
		maxX = max(maxX, p.X)
		minY = min(minY, p.Y)
		maxY = max(maxY, p.Y)
	}

	return point.Point{X: minX, Y: minY}, point.Point{X: maxX, Y: maxY}
}

// CalculateMaxDistance считает дистанцию между точками прямоугольника, описывающего комнату
func (r *Room) CalculateMaxDistance() float64 {
	p1, p2 := r.GetBoundaries()
	return point.CalculatePointsDistance(p1, p2)
}

// IsPointInRoom проверяет, находится ли точка в комнате
func (r *Room) IsPointInRoom(targetPoint point.Point) bool {
	if len(r.Area) < 3 {
		return false
	}

	return point.IsPointInPolygon(targetPoint, r.Area)
}

// IsWallBetweenPoints проверяет, есть ли стены между двумя точками.
func (a *Apartment) IsWallBetweenPoints(A, B point.Point) bool {
	segAB := &point.Segment{From: A, To: B}

	for _, wall := range a.Walls {
		x1, y1, x2, y2 := wall.Points[0].X, wall.Points[0].Y, wall.Points[1].X, wall.Points[1].Y
		if A.IsInInterval(x1, y1, x2, y2) && B.IsInInterval(x1, y1, x2, y2) {
			continue
		}

		wallSeg := &point.Segment{From: wall.Points[0], To: wall.Points[1]}
		if point.IsSegmentsIntersect(segAB, wallSeg) {
			return true
		}
	}

	return false
}

func (a *Apartment) GetEntryDoors(room *Room) ([]*Door, error) {
	entryDoors := make([]*Door, 0)
	for _, dID := range room.Doors {
		door, ok := a.doorsByID[dID]
		if !ok {
			return nil, fmt.Errorf("Failed to find door with index %s", dID)
		}

		if len(door.Rooms) == 1 {
			entryDoors = append(entryDoors, door)
		}
	}

	return entryDoors, nil
}
