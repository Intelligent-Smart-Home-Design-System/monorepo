package apartment

import (
	"fmt"
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

// GetBestCameraPoint возвращает лучшую по алгоритму точку в комнате для камеры.
// В прихожей камера ставится напротив входной двери.
// В остальных комнатах камера ставится в том месте, в котором охватывается наибольшая площадь комнаты
func (r *Room) GetBestCameraPoint(apartment *Apartment) (*point.Point, error) {
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
func (r *Room) GetBestHallCameraPoint(frontDoor *Door) (*point.Point, error) {
	doorCenter := GetObjectCenter(frontDoor.Points)

	bestPoint := r.Area[0]
	maxDist := CalculatePointsDistance(doorCenter, bestPoint)

	for _, p := range r.Area[1:] {
		dist := CalculatePointsDistance(doorCenter, p)
		if dist > maxDist {
			maxDist = dist
			bestPoint = p
		}
	}

	return &bestPoint, nil
}

// GetMaxAreaCameraPoint возвращает точку для камеры с наибольшей охватываемой площадью комнаты
func (r *Room) GetMaxAreaCameraPoint(apartment *Apartment) (*point.Point, error) {
	bestCameraPoint := r.Area[0]
	degreePerDirection := 10

	_, maxAreaCoverage, err := r.GetOptimizedCameraAreaCoverage(apartment, bestCameraPoint, degreePerDirection)
	if err != nil {
		return nil, err
	}

	for _, p := range r.Area[1:] {
		_, areaCoverage, err := r.GetOptimizedCameraAreaCoverage(apartment, p, degreePerDirection)
		if err != nil {
			return nil, err
		}

		if areaCoverage > maxAreaCoverage {
			maxAreaCoverage = areaCoverage
			bestCameraPoint = p
		}
	}

	return &bestCameraPoint, nil
}

// GetOptimizedCameraAreaCoverage вычисляет по позиции камеры наилучшее из заданных направление
// и возвращает наилучшую долю покрытия комнаты
func (r *Room) GetOptimizedCameraAreaCoverage(apartment *Apartment, cameraPoint point.Point, degreePerDirection int) (*point.Point, float64, error) {
	directionCntFloat := float64(360 / degreePerDirection)
	directionCntInt := int(directionCntFloat)
	bestDirection := point.Point{X: 1, Y: 0}

	bestCoverage, err := r.CalculateAreaCoverage(apartment, cameraPoint, bestDirection)
	if err != nil {
		return nil, 0, err
	}

	for i := 1; i < directionCntInt; i++ {
		angle := 2 * math.Pi * float64(i) / directionCntFloat
		direction := point.Point{
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
func (r *Room) CalculateAreaCoverage(apartment *Apartment, cameraPoint point.Point, cameraDirection point.Point) (float64, error) {
	gridPoints, err := r.GenerateGridPoints(0.1)
	if err != nil {
		return 0, err
	}

	if len(gridPoints) == 0 {
		return 0, nil
	}

	var cameraAngle float64 = 90 // угол обзора камеры (поставил по дефолту 90 градусов)
	visiblePoints := 0

	for _, p := range gridPoints {
		if r.IsPointVisibleOnCamera(apartment, p, cameraPoint, cameraAngle, cameraDirection) {
			visiblePoints++
		}
	}

	return float64(visiblePoints) / float64(len(gridPoints)), nil
}

// GetCameraViewDirection определяет направление камеры в комнате.
// По алгоритму камера всегда смотрит в центр комнаты,
// так как решил, что обычно это оптимальное направление
func (r *Room) GetCameraToRoomCenterDirection(cameraPoint point.Point) (*point.Point, error) {
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

	return &point.Point{X: diffX / vecLen, Y: diffY / vecLen}, nil
}

// IsPointVisibleOnCamera проверяет, видна ли точка на камере
func (r *Room) IsPointVisibleOnCamera(apartment *Apartment, p point.Point, cameraPoint point.Point, cameraAngle float64, cameraDirection point.Point) bool {
	vectorCameraToPoint := point.NewVector(cameraPoint, p)

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

	return !apartment.IsWallBetweenPoints(p, cameraPoint)
}

// GetCameraOffset определяет сдвиг для камеры
// (чтобы после корректно использовать проверку стены между точками)
func (r *Room) GetCameraOffset(apartment *Apartment, cameraPoint point.Point, cameraDirection point.Point) float64 {
	minWallLen := math.MaxFloat64

	for _, wallID := range r.Walls {
		wall := apartment.wallsByID[wallID]
		minWallLen = min(minWallLen, CalculatePointsDistance(wall.Points[0], wall.Points[1]))
	}

	return minWallLen * 0.01
}
