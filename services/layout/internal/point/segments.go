package point

import "math"

type Segment struct {
	From Point // Начальная точка отрезка в пространстве (X, Y)
	To   Point // Конечная точка отрезка в пространстве (X, Y)
}

func (s Segment) Length() float64 {
	return CalculatePointsDistance(s.From, s.To)
}

// Direction возвращает единичный вектор направления отрезка.
func (s Segment) Direction() Point {
	l := s.Length()
	return Point{
		X: (s.To.X - s.From.X) / l,
		Y: (s.To.Y - s.From.Y) / l,
	}
}

// IsSegmentsIntersect проверяет, пересекаются ли два отрезка в 2D пространстве.
// Алгоритм основан на знаках векторных произведений и корректно обрабатывает
// любые коллинеарные и параллельные случаи.
func IsSegmentsIntersect(a, b *Segment) bool {
	if a == nil || b == nil {
		return false
	}

	p1, q1 := a.From, a.To
	p2, q2 := b.From, b.To

	o1 := Orientation(p1, q1, p2)
	o2 := Orientation(p1, q1, q2)
	o3 := Orientation(p2, q2, p1)
	o4 := Orientation(p2, q2, q1)

	if o1 != o2 && o3 != o4 {
		return true
	}

	if o1 == 0 && onSegment(p1, p2, q1) {
		return true
	}
	if o2 == 0 && onSegment(p1, q2, q1) {
		return true
	}
	if o3 == 0 && onSegment(p2, p1, q2) {
		return true
	}
	if o4 == 0 && onSegment(p2, q1, q2) {
		return true
	}

	return false
}

func onSegment(p, q, r Point) bool {
	return q.X <= math.Max(p.X, r.X) && q.X >= math.Min(p.X, r.X) &&
		q.Y <= math.Max(p.Y, r.Y) && q.Y >= math.Min(p.Y, r.Y)
}

// NormalVectors возвращает два вектора нормали к отрезку s (верхний левый и нижний правый)
func (s *Segment) NormalVectors() (*Point, *Point) {
	dx := s.To.X - s.From.X
	dy := s.To.Y - s.From.Y

	return &Point{X: -dy, Y: dx}, &Point{X: dy, Y: -dx}
}
