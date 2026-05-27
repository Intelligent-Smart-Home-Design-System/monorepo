package point

// IsPointInPolygon определяет, находится ли точка p внутри произвольного полигона.
// Полигон задается слайсом указателей на точки, где вершины идут последовательно по контуру.
// Контур автоматически замыкается (последняя точка соединяется с первой).
//
// Алгоритм: Трассировка луча (Ray-Casting).
func IsPointInPolygon(p Point, polygon []*Point) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}

	inside := false

	j := n - 1
	for i := 0; i < n; i++ {
		curr := polygon[i]
		prev := polygon[j]

		if (curr.Y > p.Y) != (prev.Y > p.Y) {
			intersectX := (prev.X-curr.X)*(p.Y-curr.Y)/(prev.Y-curr.Y) + curr.X
			if p.X < intersectX {
				inside = !inside
			}
		}
		j = i
	}

	return inside
}
