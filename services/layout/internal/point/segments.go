package point

import "math"

const epsilon = 1e-9

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

func IsPointOnSegment(p, q, r Point) bool {
	PR := Point{X: r.X - p.X, Y: r.Y - p.Y}
	PQ := Point{X: q.X - p.X, Y: q.Y - p.Y}

	if math.Abs(PR.VecProduct(PQ)) <= epsilon {
		return onSegment(p, q, r)
	}

	return false
}

func ClosestPointOnSegment(p Point, seg Segment) Point {
	segmentVector := Point{
		X: seg.To.X - seg.From.X,
		Y: seg.To.Y - seg.From.Y,
	}
	pVector := Point{
		X: p.X - seg.From.X,
		Y: p.Y - seg.From.Y,
	}

	if segmentVector.X == 0 && segmentVector.Y == 0 {
        return seg.From
    }

    
    dot := segmentVector.X * pVector.X + segmentVector.Y * pVector.Y
    segmentVectorSize := segmentVector.X * segmentVector.X + segmentVector.Y * segmentVector.Y
    t := dot / segmentVectorSize
    
    if t < 0 {
        t = 0
    }
    if t > 1 {
        t = 1
    }
    
    return Point{
        X: seg.From.X + t * segmentVector.X,
        Y: seg.From.Y + t * segmentVector.Y,
    }
}
