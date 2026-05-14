package apartment

import (
	"fmt"
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

// GetCenter возвращает центр комнаты
func (r *Room) GetCenter() (*point.Point, error) {
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

		vecAB := point.NewVector(A, B)
		vecAC := point.NewVector(A, C)
		area := vecAB.VecProduct(vecAC) / 2
		totalArea += area

		centerX += (A.X + B.X + C.X) / 3 * area
		centerY += (A.Y + B.Y + C.Y) / 3 * area
	}

	centerX /= totalArea
	centerY /= totalArea

	return &point.Point{X: centerX, Y: centerY}, nil
}

// GetObjectCenter возвращает центр объекта.
// В рамках модуля объектом является то, что описано начальной и конечной точками.
// Например, дверь, окно, стена и тд.
func GetObjectCenter(points []point.Point) point.Point {
	if len(points) == 1 {
		return points[0]
	}

	return point.Point{X: (points[0].X + points[1].X) / 2, Y: (points[0].Y + points[1].Y) / 2}
}

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
	return CalculatePointsDistance(r.GetBoundaries())
}

// GenerateGridPoints генерирует сетку в комнате с заданным шагом.
// Эта функция нужна для того, чтобы проверять уровень охватываемости
// комнаты (камерой) по доле видимых точек из сетки (что удобнее)
func (r *Room) GenerateGridPoints(step float64) ([]point.Point, error) {
	points := make([]point.Point, 0)

	if len(r.Area) == 0 {
		return nil, fmt.Errorf("no corner points in room")
	}

	minPoint, maxPoint := r.GetBoundaries()

	for x := minPoint.X + step/2; x <= maxPoint.X; x += step {
		for y := minPoint.Y + step/2; y <= maxPoint.Y; y += step {
			point := point.Point{X: x, Y: y}
			if r.IsPointInRoom(point) {
				points = append(points, point)
			}
		}
	}

	return points, nil
}

// IsPointInRoom проверяет, находится ли точка в комнате
func (r *Room) IsPointInRoom(targetPoint point.Point) bool {
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

	maxCoordPoint := point.Point{X: maxAreaX + 1, Y: targetPoint.Y}
	targetSegment := point.Segment{LeftPoint: targetPoint, RightPoint: maxCoordPoint}
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
func (a *Apartment) IsWallBetweenPoints(A, B point.Point) bool {
	for _, wall := range a.Walls {
		x1, y1, x2, y2 := wall.Points[0].X, wall.Points[0].Y, wall.Points[1].X, wall.Points[1].Y
		if A.IsInSegment(x1, y1, x2, y2) && B.IsInSegment(x1, y1, x2, y2) {
			continue
		}

		segmentAB := point.Segment{LeftPoint: A, RightPoint: B}
		if segmentAB.HasSegmentIntersection(wall.Points[0], wall.Points[1]) {
			return true
		}
	}

	return false
}

// CalculatePointsDistance вычисляет расстояние между двумя точками
func CalculatePointsDistance(p1, p2 point.Point) float64 {
	diffX := p1.X - p2.X
	diffY := p1.Y - p2.Y
	return math.Sqrt(diffX * diffX + diffY * diffY)
}

// MovePointInDirection сдвигает вектор по направлению в offset раз
func MovePointInDirection(vec point.Point, vecDirection point.Point, offset float64) point.Point {
	return point.Point{
		X: vec.X + vecDirection.X * offset,
		Y: vec.Y + vecDirection.Y * offset,
	}
}

