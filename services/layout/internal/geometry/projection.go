package geometry

import (
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

// ProjectPointToSegment проецирует точку на ось отрезка.
// Возвращает координату вдоль отрезка (0..length).
// Не зажимает, позволяет caller'у решать, нужен ли clamp.
func ProjectPointToSegment(seg point.Segment, p point.Point) float64 {
	dir := seg.Direction()
	dx := p.X - seg.From.X
	dy := p.Y - seg.From.Y
	return dx*dir.X + dy*dir.Y
}

// ProjectPolygonToSegment проецирует произвольный полигон на ось отрезка и обрезает результат по его границам.
//
// Возвращает два значения:
//  1. *point.Interval — одномерный отрезок [From, To] вдоль локальной оси (зажат в диапазоне от 0 до length).
//  2. *point.Segment  — двумерный отрезок в пространстве, состоящий из двух реальных вершин полигона,
//     которые дали минимальную и максимальную проекции.
//
// Если слайс точек пуст или проекция полигона полностью лежит за пределами отрезка, возвращает (nil, nil).
func ProjectPolygonToSegment(seg point.Segment, points []point.Point) (*point.Interval, *point.Segment) {
	if len(points) == 0 {
		return nil, nil
	}

	minProj := math.MaxFloat64
	maxProj := -math.MaxFloat64
	var polygonSides point.Segment

	for _, p := range points {
		proj := ProjectPointToSegment(seg, p)
		if minProj >= proj {
			minProj = proj
			polygonSides.To = p
		}
		if maxProj <= proj {
			maxProj = proj
			polygonSides.From = p
		}
	}

	// проекция полностью за пределами отрезка
	segLen := seg.Length()
	if maxProj < 0 || minProj > segLen {
		return nil, nil
	}

	// зажимаем в границы отрезка
	return &point.Interval{
		From: math.Max(0, minProj),
		To:   math.Min(segLen, maxProj),
	}, &polygonSides
}

// IntervalToPoints преобразует локальный сегмент [From, To]
// обратно в две точки (X, Y) в обычной декартовой плоскости.
func IntervalToPoints(seg point.Segment, interval *point.Interval) []point.Point {
	if interval == nil {
		return nil
	}

	dir := seg.Direction()
	startPoint := seg.From

	pFrom := point.MovePointInDirection(startPoint, dir, interval.From)
	pTo := point.MovePointInDirection(startPoint, dir, interval.To)

	return []point.Point{pFrom, pTo}
}
