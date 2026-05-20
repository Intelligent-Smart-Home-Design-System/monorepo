package point

import "math"

// VecProduct - векторное произведение
func (p *Point) VecProduct(vector Point) float64 {
	return p.X * vector.Y - p.Y * vector.X
}

// HasSegmentIntersection проверяет, пересекаются ли два отрезка.
// Первый отрезок - это s, второй отрезок - это AB
func (s *Segment) HasSegmentIntersection(A, B Point) bool {
	C := s.LeftPoint
	D := s.RightPoint

	vecAB := NewVector(A, B)
	vecAC := NewVector(A, C)
	pr1 := vecAB.VecProduct(vecAC)

	vecAD := NewVector(A, D)
	pr2 := vecAB.VecProduct(vecAD)

	vecCD := NewVector(C, D)
	vecCA := NewVector(C, A)
	pr3 := vecCD.VecProduct(vecCA)

	vecCB := NewVector(C, B)
	pr4 := vecCD.VecProduct(vecCB)

	if pr1 == 0 || pr2 == 0 || pr3 == 0 || pr4 == 0 {
		if (max(A.X, B.X) < min(C.X, D.X)) || (max(C.X, D.X) < min(A.X, B.X)) {
			return false
		}

		if (max(A.Y, B.Y) < min(C.Y, D.Y)) || (max(C.Y, D.Y) < min(A.Y, B.Y)) {
			return false
		}
	}

	if (pr1 > 0 && pr2 > 0) || (pr1 < 0 && pr2 < 0) {
		return false
	}

	if (pr3 > 0 && pr4 > 0) || (pr3 < 0 && pr4 < 0) {
		return false
	}

	if max(A.Y, B.Y) == A.Y {
		if pr3 == 0 {
			return false
		}
	} else {
		if pr4 == 0 {
			return false
		}
	}

	return true
}

// IsInSegment проверяет, находится ли точка на отрезке
func (p *Point) IsInSegment(x1, y1, x2, y2 float64) bool {
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
	return math.Sqrt(diffX * diffX + diffY * diffY)
}

// MovePointInDirection сдвигает вектор по направлению в offset раз
func MovePointInDirection(vec Point, vecDirection Point, offset float64) Point {
	return Point{
		X: vec.X + vecDirection.X * offset,
		Y: vec.Y + vecDirection.Y * offset,
	}
}
