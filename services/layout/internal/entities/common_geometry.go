package entities

import (
	"fmt"
	"math"
)

// GetCenter возвращает центр комнаты
func (r *Room) GetCenter() (*Point, error) {
	if len(r.Area) < 3 {
		return nil, fmt.Errorf("need al least 3 points in room")
	}

	var totalArea float64 = 0
	var centerX float64 = 0
	var centerY float64 = 0

	A := r.Area[0]

	for i := 2; i < len(r.Area); i++ {
		B := r.Area[i-1]
		C := r.Area[i]

		vecAB := NewVector(A, B)
		vecAC := NewVector(A, C)
		area := vecAB.VecProduct(vecAC) / 2
		totalArea += area

		centerX += (A.X + B.X + C.X) / 3 * area
		centerY += (A.Y + B.Y + C.Y) / 3 * area
	}

	centerX /= totalArea
	centerY /= totalArea

	return &Point{centerX, centerY}, nil
}

// VecProduct - векторное произведение
func (p *Point) VecProduct(vector Point) float64 {
	return p.X * vector.Y - p.Y * vector.X
}

// GetObjectCenter возвращает центр объекта.
// В рамках модуля объектом является то, что описано начальной и конечной точками.
// Например, дверь, окно, стена и тд.
func GetObjectCenter(points []Point) Point {
	if len(points) == 1 {
		return points[0]
	}

	return Point{(points[0].X + points[1].X) / 2, (points[0].Y + points[1].Y) / 2}
}

// GenerateGridPoints генерирует сетку в комнате с заданным шагом.
// Эта функция нужна для того, чтобы проверять уровень охватываемости
// комнаты (камерой) по доле видимых точек из сетки (что удобнее)
func (r *Room) GenerateGridPoints(step float64) ([]Point, error) {
	points := make([]Point, 0)

	if len(r.Area) == 0 {
		return nil, fmt.Errorf("no corner points in room")
	}

	minX, minY, maxX, maxY := r.Area[0].X, r.Area[0].Y, r.Area[0].X, r.Area[0].Y
	for _, point := range r.Area[1:] {
		minX = min(minX, point.X)
		maxX = max(maxX, point.X)
		minY = min(minY, point.Y)
		maxY = max(maxY, point.Y)
	}

	minPoint, maxPoint := Point{minX, minY}, Point{maxX, maxY}

	for x := minPoint.X + step/2; x <= maxPoint.X; x += step {
		for y := minPoint.Y + step/2; y <= maxPoint.Y; y += step {
			point := Point{x, y}
			if r.IsPointInRoom(point) {
				points = append(points, point)
			}
		}
	}

	return points, nil
}

// IsPointInRoom проверяет, находится ли точка в комнате
func (r *Room) IsPointInRoom(targetPoint Point) bool {
	points := r.Area
	maxAreaX := points[0].X
	if targetPoint.X == points[0].X && targetPoint.Y == points[0].Y {
		return true
	}

	for _, point := range points[1:] {
		maxAreaX = max(maxAreaX, point.X)
		if targetPoint.X == point.X && targetPoint.Y == point.Y {
			return true
		}
	}

	maxCoordPoint := Point{maxAreaX + 1, targetPoint.Y}
	targetSegment := Segment{targetPoint, maxCoordPoint}
	intersectionCnts := 0

	n := len(points)
	for i := 1; i < n; i++ {
		if targetPoint.IsInSegment(points[i-1].X, points[i-1].Y, points[i].X, points[i].Y) {
			return true
		}

		if targetSegment.HasSegmentIntersection(points[i-1], points[i]) {
			intersectionCnts++
		}
	}

	if targetPoint.IsInSegment(points[n-1].X, points[n-1].Y, points[0].X, points[0].Y) {
		return true
	}

	if targetSegment.HasSegmentIntersection(points[n-1], points[0]) {
		intersectionCnts++
	}

	return intersectionCnts%2 != 0
}

// IsWallBetweenPoints проверяет, есть ли стены между двумя точками
func (a *Apartment) IsWallBetweenPoints(A, B Point) bool {
	for _, wall := range a.Walls {
		x1, y1, x2, y2 := wall.Points[0].X, wall.Points[0].Y, wall.Points[1].X, wall.Points[1].Y
		if A.IsInSegment(x1, y1, x2, y2) && B.IsInSegment(x1, y1, x2, y2) {
			continue
		}

		segmentAB := Segment{A, B}
		if segmentAB.HasSegmentIntersection(wall.Points[0], wall.Points[1]) {
			return true
		}
	}

	return false
}

// CalculatePointsDistance вычисляет расстояние между двумя точками
func CalculatePointsDistance(p1, p2 Point) float64 {
	diffX := p1.X - p2.X
	diffY := p1.Y - p2.Y
	return math.Sqrt(diffX * diffX + diffY * diffY)
}

// MovePointInDirection сдвигает вектор по направлению в offset раз
func MovePointInDirection(vec Point, vecDirection Point, offset float64) Point {
	return Point{
		X: vec.X + vecDirection.X * offset,
		Y: vec.Y + vecDirection.Y * offset,
	}
}

