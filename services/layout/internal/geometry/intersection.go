package geometry

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

// IsSegmentIntersectPolygon проверяет, пересекает ли отрезок полигон.
func IsSegmentIntersectPolygon(polygon []*point.Point, seg point.Segment) bool {
	n := len(polygon)
	for i := 0; i < n; i++ {
		side := point.Segment{
			From: *polygon[i],
			To:   *polygon[(i+1)%n],
		}
		if point.IsSegmentsIntersect(&side, &seg) {
			return true
		}
	}

	if point.IsPointInPolygon(seg.From, polygon) || point.IsPointInPolygon(seg.To, polygon) {
		return true
	}

	return false
}
