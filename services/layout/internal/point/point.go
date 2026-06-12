package point

import "math"

type Point struct {
	X float64
	Y float64
}

func NewVector(p1, p2 Point) Point {
	return Point{p2.X - p1.X, p2.Y - p1.Y}
}

// Orientation определяет взаимное расположение трех точек (p, q, r).
// Возвращает:
//
//	0 -> точки лежат на одной прямой (коллинеарны)
//	1 -> по часовой стрелке
//	2 -> против часовой стрелки
func Orientation(p, q, r Point) int {
	val := (q.Y-p.Y)*(r.X-q.X) - (q.X-p.X)*(r.Y-q.Y)
	if val == 0 {
		return 0
	}
	if val > 0 {
		return 1
	}
	return 2
}

func PointToSquare(p Point, shift float64) []Point {
	return []Point{
		{X: p.X - shift, Y: p.Y - shift},
		{X: p.X - shift, Y: p.Y + shift},
		{X: p.X + shift, Y: p.Y + shift},
		{X: p.X + shift, Y: p.Y - shift},
	}
}

// GetDirectionToPoint возвращает направление от точки p к точке q.
// Функция нормирует к единичному вектору
func GetDirectionToPoint(p, q Point) Point {
	dx := q.X - p.X
	dy := q.Y - p.Y

	size := math.Sqrt(dx * dx + dy * dy)
	if size == 0 {
		return Point{X: 1, Y: 0}
	}

	return Point{X: dx / size, Y: dy / size}
}
