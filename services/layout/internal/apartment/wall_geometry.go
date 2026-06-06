package apartment

import (
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

func (w Wall) Length() float64 {
	start := w.Points[0]
	end := w.Points[1]
	dx := end.X - start.X
	dy := end.Y - start.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (w Wall) Direction() point.Point {
	l := w.Length()

	start := w.Points[0]
	end := w.Points[1]

	return point.Point{
		X: (end.X - start.X) / l,
		Y: (end.Y - start.Y) / l,
	}
}
