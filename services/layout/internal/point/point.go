package point

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
