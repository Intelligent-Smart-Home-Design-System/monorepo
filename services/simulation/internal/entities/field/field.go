package field

import (
	//"encoding/json"
	//"fmt"
	"math"
	//"os"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
)

//===Вспомогательные геометрические функции===
// PointInRoom проверяет находится ли точка внутри комнаты методом ray casting.
func PointInRoom(x, y float64, room api.Room) bool {
	return pointInPolygon(x, y, room.Area)
}
 
// PolygonIntersectsCircle возвращает true если окружность пересекает многоугольник.
func PolygonIntersectsCircle(polygon [][2]float64, cx, cy, radius float64) bool {
	if pointInPolygon(cx, cy, polygon) {
		return true
	}
	r2 := radius * radius
	for _, v := range polygon {
		dx := v[0] - cx
		dy := v[1] - cy
		if dx*dx+dy*dy <= r2 {
			return true
		}
	}
	n := len(polygon)
	for i := 0; i < n; i++ {
		a := polygon[i]
		b := polygon[(i+1)%n]
		if segmentIntersectsCircle(a, b, cx, cy, radius) {
			return true
		}
	}
	return false
}
 
func segmentIntersectsCircle(a, b [2]float64, cx, cy, radius float64) bool {
	dx := b[0] - a[0]
	dy := b[1] - a[1]
	fx := a[0] - cx
	fy := a[1] - cy
 
	A := dx*dx + dy*dy
	B := 2 * (fx*dx + fy*dy)
	C := fx*fx + fy*fy - radius*radius
 
	discriminant := B*B - 4*A*C
	if discriminant < 0 {
		return false
	}
	disc := math.Sqrt(discriminant)
	t1 := (-B - disc) / (2 * A)
	t2 := (-B + disc) / (2 * A)
	return (t1 >= 0 && t1 <= 1) || (t2 >= 0 && t2 <= 1)
}
 
func pointInPolygon(x, y float64, polygon [][2]float64) bool {
	n := len(polygon)
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := polygon[i][0], polygon[i][1]
		xj, yj := polygon[j][0], polygon[j][1]
		if ((yi > y) != (yj > y)) &&
			(x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}