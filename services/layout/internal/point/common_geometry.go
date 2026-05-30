package point

import (
	"math"
)

// VecProduct - векторное произведение
func (p *Point) VecProduct(vector Point) float64 {
	return p.X*vector.Y - p.Y*vector.X
}

// TODO x1, y1, x2, y2 float64 переместить в сегмент
// IsInInterval проверяет, находится ли точка на отрезке
func (p *Point) IsInInterval(x1, y1, x2, y2 float64) bool {
	return p.IsInRay(x1, y1, x2, y2) && p.IsInRay(x2, y2, x1, y1)
}

// IsInRay проверяет, лежит ли точка на луче
// (начало луча - точка (x1, y1), направление - вектор ((x2 - x1), (y2 - y1)))
func (p *Point) IsInRay(x1, y1, x2, y2 float64) bool {
	A := y2 - y1
	B := -(x2 - x1)
	C := -x1*(y2-y1) + y1*(x2-x1)

	if A*p.X+B*p.Y+C != 0 {
		return false
	}

	if x1 != p.X {
		if ((x1 - x2) / (x1 - p.X)) < 0 {
			return false
		}
	}

	if y1 != p.Y {
		if ((y1 - y2) / (y1 - p.Y)) < 0 {
			return false
		}
	}

	return true
}

// CalculatePointsDistance вычисляет расстояние между двумя точками
func CalculatePointsDistance(p1, p2 Point) float64 {
	diffX := p1.X - p2.X
	diffY := p1.Y - p2.Y
	return math.Sqrt(diffX*diffX + diffY*diffY)
}

// MovePointInDirection сдвигает точку/вектор по направлению в offset раз
func MovePointInDirection(p Point, direction Point, offset float64) Point {
	size := math.Sqrt(direction.X * direction.X + direction.Y * direction.Y)
	if size == 0 {
		return p
	}

	return Point{
		X: p.X + (direction.X / size) * offset,
		Y: p.Y + (direction.Y / size) * offset,
	}
}

// MovePointInDirectionPlusOffset сдвигает точку/вектор по направлению на offset раз
func MovePointInDirectionPlusOffset(p Point, direction Point, offset float64) Point {
	size := math.Sqrt(direction.X * direction.X + direction.Y * direction.Y)
	if size == 0 {
		return p
	}

	total := size + offset 

	return Point{
		X: p.X + (direction.X / size) * total,
		Y: p.Y + (direction.Y / size) * total,
	}
}
