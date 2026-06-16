package point

import (
	"encoding/json"
	"math"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (p *Point) UnmarshalJSON(data []byte) error {
	var coords [2]float64
	
	if err := json.Unmarshal(data, &coords); err != nil {
		return err
	}
	
	p.X = coords[0]
	p.Y = coords[1]
	
	return nil
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

	size := math.Sqrt(dx*dx + dy*dy)
	if size == 0 {
		return Point{X: 1, Y: 0}
	}

	return Point{X: dx / size, Y: dy / size}
}

func Normalize(vector Point) Point {
	size := math.Sqrt(vector.X*vector.X + vector.Y*vector.Y)
	if size == 0 {
		return Point{X: 0, Y: 1}
	}

	return Point{X: vector.X / size, Y: vector.Y / size}
}
