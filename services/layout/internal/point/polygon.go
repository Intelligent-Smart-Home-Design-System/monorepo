package point

import "fmt"

// IsPointInPolygon определяет, находится ли точка p внутри произвольного полигона.
// Полигон задается слайсом точек, где вершины идут последовательно по контуру.
// Контур автоматически замыкается (последняя точка соединяется с первой).
//
// Алгоритм: Трассировка луча (Ray-Casting).
func IsPointInPolygon(p Point, polygon []Point) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}

	inside := false

	j := n - 1
	for i := range n {
		curr := polygon[i]
		prev := polygon[j]

		if onSegment(curr, p, prev) {
			return true
		}

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

// GetCenter находит центр полигона
func GetCenter(polygon []Point) *Point {
	if len(polygon) < 3 {
		center := GetObjectCenter(polygon)
		return &center
	}
	var totalArea float64 = 0
	var centerX float64 = 0
	var centerY float64 = 0

	A := polygon[0]

	for i := 2; i < len(polygon); i++ {
		B := polygon[i-1]
		C := polygon[i]

		vecAB := NewVector(A, B)
		vecAC := NewVector(A, C)
		area := vecAB.VecProduct(vecAC) / 2
		totalArea += area

		centerX += (A.X + B.X + C.X) / 3 * area
		centerY += (A.Y + B.Y + C.Y) / 3 * area
	}

	centerX /= totalArea
	centerY /= totalArea
	
	return &Point{X: centerX, Y: centerY}
}

// GetObjectCenter возвращает центр объекта.
// В рамках модуля объектом является то, что описано начальной и конечной точками.
// Например, дверь, окно, стена и тд.
func GetObjectCenter(polygon []Point) Point {
	if len(polygon) == 1 {
		return polygon[0]
	}

	return Point{X: (polygon[0].X + polygon[1].X) / 2, Y: (polygon[0].Y + polygon[1].Y) / 2}
}

// GetBoundaries возвращает противоположные точки полигона
func GetBoundaries(polygon []Point) (Point, Point) {
	minX, minY, maxX, maxY := polygon[0].X, polygon[0].Y, polygon[0].X, polygon[0].Y
	for _, p := range polygon[1:] {
		minX = min(minX, p.X)
		maxX = max(maxX, p.X)
		minY = min(minY, p.Y)
		maxY = max(maxY, p.Y)
	}

	return Point{X: minX, Y: minY}, Point{X: maxX, Y: maxY}
}

// CalculateMaxDistance считает максимальную дистанцию между точками полигона
func CalculateMaxDistance(polygon []Point) float64 {
	p1, p2 := GetBoundaries(polygon)
	return CalculatePointsDistance(p1, p2)
}

// GenerateGridPoints генерирует сетку в комнате с заданным шагом.
// Эта функция нужна для того, чтобы проверять уровень охватываемости
// комнаты (камерой) по доле видимых точек из сетки (что удобнее)
func GenerateGridPoints(polygon []Point, step float64) ([]Point, error) {
	result := make([]Point, 0)

	if len(polygon) == 0 {
		return nil, fmt.Errorf("no corner points in room")
	}

	minPoint, maxPoint := GetBoundaries(polygon)

	for x := minPoint.X + step/2; x <= maxPoint.X; x += step {
		for y := minPoint.Y + step/2; y <= maxPoint.Y; y += step {
			p := Point{X: x, Y: y}
			if IsPointInPolygon(p, polygon) {
				result = append(result, p)
			}
		}
	}

	return result, nil
}
