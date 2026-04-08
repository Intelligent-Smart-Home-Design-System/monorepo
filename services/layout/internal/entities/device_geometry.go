package entities

import (
	"fmt"
	"math"
)

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
	bestCameraPoint := r.Area[0]
	degreePerDirection := 10

	_, maxAreaCoverage, err := r.GetOptimizedCameraAreaCoverage(apartment, bestCameraPoint, degreePerDirection)
	if err != nil {
		return nil, err
	}

	for _, point := range r.Area[1:] {
		_, areaCoverage, err := r.GetOptimizedCameraAreaCoverage(apartment, point, degreePerDirection)
		if err != nil {
			return nil, err
		}

		if areaCoverage > maxAreaCoverage {
			maxAreaCoverage = areaCoverage
			bestCameraPoint = point
		}
	}

	return &bestCameraPoint, nil
}

// GetOptimizedCameraAreaCoverage вычисляет по позиции камеры наилучшее из заданных направление
// и возвращает наилучшую долю покрытия комнаты
func (r *Room) GetOptimizedCameraAreaCoverage(apartment *Apartment, cameraPoint Point, degreePerDirection int) (*Point, float64, error) {
	directionCntFloat := float64(360 / degreePerDirection)
	directionCntInt := int(directionCntFloat)
	bestDirection := Point{X: 1, Y: 0}

	bestCoverage, err := r.CalculateAreaCoverage(apartment, cameraPoint, bestDirection)
	if err != nil {
		return nil, 0, err
	}

	for i := 1; i < directionCntInt; i++ {
		angle := 2 * math.Pi * float64(i) / directionCntFloat
		direction := Point{
			X: math.Cos(angle),
			Y: math.Sin(angle),
		}

		coverage, err := r.CalculateAreaCoverage(apartment, cameraPoint, direction)
		if err != nil {
			return nil, 0, err
		}

		if coverage > bestCoverage {
			bestCoverage = coverage
			bestDirection = direction
		}
	}

	return &bestDirection, bestCoverage, nil
}

// CalculateAreaCoverage вычисляет охватываемую площадь комнаты по заданной точке для камеры
func (r *Room) CalculateAreaCoverage(apartment *Apartment, cameraPoint Point, cameraDirection Point) (float64, error) {
	gridPoints, err := r.GenerateGridPoints(0.1)
	if err != nil {
		return 0, err
	}

	if len(gridPoints) == 0 {
		return 0, nil
	}

	var cameraAngle float64 = 90 // угол обзора камеры (поставил по дефолту 90 градусов)
	visiblePoints := 0

	for _, point := range gridPoints {
		if r.IsPointVisibleOnCamera(apartment, point, cameraPoint, cameraAngle, cameraDirection) {
			visiblePoints++
		}
	}

	return float64(visiblePoints) / float64(len(gridPoints)), nil
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

// GetCameraViewDirection определяет направление камеры в комнате.
// По алгоритму камера всегда смотрит в центр комнаты,
// так как решил, что обычно это оптимальное направление
func (r *Room) GetCameraToRoomCenterDirection(cameraPoint Point) (*Point, error) {
	roomCenter, err := r.GetCenter()
	if err != nil {
		return nil, err
	}

	diffX := roomCenter.X - cameraPoint.X
	diffY := roomCenter.Y - cameraPoint.Y
	vecLen := math.Sqrt(diffX*diffX + diffY*diffY)

	if vecLen == 0 {
		return nil, fmt.Errorf("Room corner is its center")
	}

	return &Point{diffX / vecLen, diffY / vecLen}, nil
}

// IsPointVisibleOnCamera проверяет, видна ли точка на камере
func (r *Room) IsPointVisibleOnCamera(apartment *Apartment, point Point, cameraPoint Point, cameraAngle float64, cameraDirection Point) bool {
	vectorCameraToPoint := NewVector(cameraPoint, point)

	if vectorCameraToPoint.X != 0 || vectorCameraToPoint.Y != 0 {
		vecLen := math.Sqrt(vectorCameraToPoint.X*vectorCameraToPoint.X + vectorCameraToPoint.Y*vectorCameraToPoint.Y)
		vectorCameraToPoint.X /= vecLen
		vectorCameraToPoint.Y /= vecLen

		dotPr := vectorCameraToPoint.X*cameraDirection.X + vectorCameraToPoint.Y*cameraDirection.Y
		angle := math.Acos(dotPr) * (180 / math.Pi)

		if angle > cameraAngle/2 {
			return false
		}
	}

	cameraOffset := r.GetCameraOffset(apartment, cameraPoint, cameraDirection)
	cameraPoint = MovePointInDirection(cameraPoint, cameraDirection, cameraOffset)

	return !apartment.IsWallBetweenPoints(point, cameraPoint)
}

// GetCameraOffset определяет сдвиг для камеры
// (чтобы после корректно использовать проверку стены между точками)
func (r *Room) GetCameraOffset(apartment *Apartment, cameraPoint Point, cameraDirection Point) float64 {
	minWallLen := math.MaxFloat64

	for _, wallID := range r.Walls {
		wall := apartment.wallsByID[wallID]
		minWallLen = min(minWallLen, CalculatePointsDistance(wall.Points[0], wall.Points[1]))
	}

	return minWallLen * 0.01
}
