package entities

import (
	"fmt"
	"math"
)

// GetCenter возвращает центр комнаты
func (r *Room) GetCenter() (*Point, error) {
	if len(r.Area) < 3 {
		return nil, fmt.Errorf("need al least 3 points in room")
	}

	var totalArea float64 = 0
	var centerX float64 = 0
	var centerY float64 = 0

	A := r.Area[0]

	for i := 2; i < len(r.Area); i++ {
		B := r.Area[i-1]
		C := r.Area[i]

		vecAB := NewVector(A, B)
		vecAC := NewVector(A, C)
		area := vecAB.VecProduct(vecAC) / 2
		totalArea += area

		centerX += (A.X + B.X + C.X) / 3 * area
		centerY += (A.Y + B.Y + C.Y) / 3 * area
	}

	centerX /= totalArea
	centerY /= totalArea

	return &Point{centerX, centerY}, nil
}

// VecProduct - векторное произведение
func (p *Point) VecProduct(vector Point) float64 {
	return p.X * vector.Y - p.Y * vector.X
}

// GetObjectCenter возвращает центр объекта.
// В рамках модуля объектом является то, что описано начальной и конечной точками.
// Например, дверь, окно, стена и тд.
func GetObjectCenter(points []Point) Point {
	if len(points) == 1 {
		return points[0]
	}

	return Point{(points[0].X + points[1].X) / 2, (points[0].Y + points[1].Y) / 2}
}

// GetBestCameraPoint возвращает лучшую по алгоритму точку в комнате для камеры.
// В прихожей камера ставится напротив входной двери.
// В остальных комнатах камера ставится в том месте, в котором охватывается наибольшая площадь комнаты
func (r *Room) GetBestCameraPoint(apartment *Apartment) (*Point, error) {
	if r.Name == "hall" {
		frontDoor := apartment.GetFrontDoor()
		if frontDoor == nil {
			return nil, fmt.Errorf("no front door in apartment")
		}

		return r.GetBestHallCameraPoint(frontDoor)
	}

	return r.GetMaxAreaCameraPoint(apartment)
}

// GetBestHallCameraPoint возвращает лучшую точку для камеры в прихожей (напротив входной двери)
func (r *Room) GetBestHallCameraPoint(frontDoor *Door) (*Point, error) {
	doorCenter := GetObjectCenter(frontDoor.Points)

	bestPoint := r.Area[0]
	maxDist := CalculatePointsDistance(doorCenter, bestPoint)

	for _, point := range r.Area[1:] {
		dist := CalculatePointsDistance(doorCenter, point)
		if dist > maxDist {
			maxDist = dist
			bestPoint = point
		}
	}

	return &bestPoint, nil
}

// GetMaxAreaCameraPoint возвращает точку для камеры с наибольшей охватываемой площадью комнаты
func (r *Room) GetMaxAreaCameraPoint(apartment *Apartment) (*Point, error) {
	bestPoint := r.Area[0]
	maxAreaCoverage, err := r.CalculateAreaCoverage(apartment, bestPoint)
	if err != nil {
		return nil, err
	}

	for _, point := range r.Area[1:] {
		areaCoverage, err := r.CalculateAreaCoverage(apartment, point)
		if err != nil {
			return nil, err
		}
		if areaCoverage > maxAreaCoverage {
			maxAreaCoverage = areaCoverage
			bestPoint = point
		}
	}

	return &bestPoint, nil
}

// CalculateAreaCoverage вычисляет охватываемую площадь комнаты по заданной точке для камеры
func (r *Room) CalculateAreaCoverage(apartment *Apartment, cameraPoint Point) (float64, error) {
	gridPoints, err := r.GenerateGridPoints(0.1)
	if err != nil {
		return 0, err
	}

	if len(gridPoints) == 0 {
		return 0, nil
	}

	cameraViewDirection, err := r.GetCameraViewDirection(cameraPoint)
	if err != nil {
		return 0, err
	}

	var cameraAngle float64 = 90 // угол обзора камеры (поставил по дефолту 90 градусов)
	visiblePoints := 0

	for _, point := range gridPoints {
		if r.IsPointVisibleOnCamera(apartment, point, cameraPoint, cameraAngle, *cameraViewDirection) {
			visiblePoints++
		}
	}

	return float64(visiblePoints) / float64(len(gridPoints)), nil
}

// GenerateGridPoints генерирует сетку в комнате с заданным шагом.
// Эта функция нужна для того, чтобы проверять уровень охватываемости
// комнаты (камерой) по доле видимых точек из сетки (что удобнее)
func (r *Room) GenerateGridPoints(step float64) ([]Point, error) {
	points := make([]Point, 0)

	if len(r.Area) == 0 {
		return nil, fmt.Errorf("no corner points in room")
	}

	minX, minY, maxX, maxY := r.Area[0].X, r.Area[0].Y, r.Area[0].X, r.Area[0].Y
	for _, point := range r.Area[1:] {
		minX = min(minX, point.X)
		maxX = max(maxX, point.X)
		minY = min(minY, point.Y)
		maxY = max(maxY, point.Y)
	}

	minPoint, maxPoint := Point{minX, minY}, Point{maxX, maxY}

	for x := minPoint.X + step/2; x <= maxPoint.X; x += step {
		for y := minPoint.Y + step/2; y <= maxPoint.Y; y += step {
			point := Point{x, y}
			if r.IsPointInRoom(point) {
				points = append(points, point)
			}
		}
	}

	return points, nil
}

// IsPointInRoom проверяет, находится ли точка в комнате
func (r *Room) IsPointInRoom(targetPoint Point) bool {
	points := r.Area
	maxAreaX := points[0].X
	if targetPoint.X == points[0].X && targetPoint.Y == points[0].Y {
		return true
	}

	for _, point := range points[1:] {
		maxAreaX = max(maxAreaX, point.X)
		if targetPoint.X == point.X && targetPoint.Y == point.Y {
			return true
		}
	}

	maxCoordPoint := Point{maxAreaX + 1, targetPoint.Y}
	targetSegment := Segment{targetPoint, maxCoordPoint}
	intersectionCnts := 0

	n := len(points)
	for i := 1; i < n; i++ {
		if targetPoint.IsInSegment(points[i-1].X, points[i-1].Y, points[i].X, points[i].Y) {
			return true
		}

		if targetSegment.HasSegmentIntersection(points[i-1], points[i]) {
			intersectionCnts++
		}
	}

	if targetPoint.IsInSegment(points[n-1].X, points[n-1].Y, points[0].X, points[0].Y) {
		return true
	}

	if targetSegment.HasSegmentIntersection(points[n-1], points[0]) {
		intersectionCnts++
	}

	return intersectionCnts%2 != 0
}

// IsInSegment проверяет, находится ли точка на отрезке
func (p *Point) IsInSegment(x1, y1, x2, y2 float64) bool {
	return p.IsInRay(x1, y1, x2, y2) && p.IsInRay(x2, y2, x1, y1)
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

// IsInRay проверяет, лежит ли точка на луче
// (начало луча - точка (x1, y1), направление - вектор ((x2 - x1), (y2 - y1)))
func (p *Point) IsInRay(x1, y1, x2, y2 float64) bool {
	A := y2 - y1
	B := -(x2 - x1)
	C := -x1 * (y2 - y1) + y1 * (x2 - x1)

	if A * p.X + B * p.Y + C != 0 {
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

// GetCameraViewDirection определяет направление камеры в комнате.
// По алгоритму камера всегда смотрит в центр комнаты,
// так как решил, что обычно это оптимальное направление
func (r *Room) GetCameraViewDirection(cameraPoint Point) (*Point, error) {
	roomCenter, err := r.GetCenter()
	if err != nil {
		return nil, err
	}

	diffX := roomCenter.X - cameraPoint.X
	diffY := roomCenter.Y - cameraPoint.Y
	vecLen := math.Sqrt(diffX * diffX + diffY * diffY)

	if vecLen == 0 {
		return nil, fmt.Errorf("Room corner is its center")
	}

	return &Point{diffX / vecLen, diffY / vecLen}, nil
}

// IsPointVisibleOnCamera проверяет, видна ли точка на камере
func (r *Room) IsPointVisibleOnCamera(apartment *Apartment, point Point, cameraPoint Point, cameraAngle float64, cameraViewDirection Point) bool {
	vectorCameraToPoint := NewVector(cameraPoint, point)

	if vectorCameraToPoint.X != 0 || vectorCameraToPoint.Y != 0 {
		vecLen := math.Sqrt(vectorCameraToPoint.X * vectorCameraToPoint.X + vectorCameraToPoint.Y * vectorCameraToPoint.Y)
		vectorCameraToPoint.X /= vecLen
		vectorCameraToPoint.Y /= vecLen

		dotPr := vectorCameraToPoint.X * cameraViewDirection.X + vectorCameraToPoint.Y * cameraViewDirection.Y
		angle := math.Acos(dotPr) * (180 / math.Pi)

		if angle > cameraAngle/2 {
			return false
		}
	}

	return !r.IsWallBetweenPoints(apartment, point, cameraPoint)
}

// IsWallBetweenPoints проверяет, есть ли стены между двумя точками
func (r *Room) IsWallBetweenPoints(apartment *Apartment, A, B Point) bool {
	for _, wall := range apartment.Walls {
		x1, y1, x2, y2 := wall.Points[0].X, wall.Points[0].Y, wall.Points[1].X, wall.Points[1].Y
		if !A.IsInSegment(x1, y1, x2, y2) && !B.IsInSegment(x1, y1, x2, y2) {
			segmentAB := Segment{A, B}
			if segmentAB.HasSegmentIntersection(wall.Points[0], wall.Points[1]) {
				return true
			}
		}
	}

	return false
}

// CalculatePointsDistance вычисляет расстояние между двумя точками
func CalculatePointsDistance(p1, p2 Point) float64 {
	diffX := p1.X - p2.X
	diffY := p1.Y - p2.Y
	return math.Sqrt(diffX * diffX + diffY * diffY)
}
