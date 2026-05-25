package apartment

import (
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const baseDeviceAngle = 100

// GetMaxAreadevicePoint возвращает точку для устройства с наибольшей охватываемой площадью комнаты
func (r *Room) GetMaxAreaDevicePoint(apartment *Apartment, deviceRange float64, deviceAngle float64) (*point.Point, error) {
	bestDevicePoint := r.Area[0]
	degreePerDirection := 10

	_, maxAreaCoverage, err := r.GetOptimizedDeviceAreaCoverage(apartment, bestDevicePoint, degreePerDirection, deviceRange, deviceAngle)
	if err != nil {
		return nil, err
	}

	for _, p := range r.Area[1:] {
		_, areaCoverage, err := r.GetOptimizedDeviceAreaCoverage(apartment, p, degreePerDirection, deviceRange, deviceAngle)
		if err != nil {
			return nil, err
		}

		if areaCoverage > maxAreaCoverage {
			maxAreaCoverage = areaCoverage
			bestDevicePoint = p
		}
	}

	return &bestDevicePoint, nil
}

// GetOptimizedDeviceAreaCoverage вычисляет по позиции устройства наилучшее из заданных направление
// и возвращает наилучшую долю покрытия комнаты
func (r *Room) GetOptimizedDeviceAreaCoverage(apartment *Apartment, devicePoint point.Point, degreePerDirection int, deviceRange float64, deviceAngle float64) (*point.Point, float64, error) {
	directionCntFloat := float64(360 / degreePerDirection)
	directionCntInt := int(directionCntFloat)
	bestDirection := point.Point{X: 1, Y: 0}

	bestCoverage, err := r.CalculateAreaCoverage(apartment, devicePoint, bestDirection, deviceRange, deviceAngle)
	if err != nil {
		return nil, 0, err
	}

	for i := 1; i < directionCntInt; i++ {
		angle := 2 * math.Pi * float64(i) / directionCntFloat
		direction := point.Point{
			X: math.Cos(angle),
			Y: math.Sin(angle),
		}

		coverage, err := r.CalculateAreaCoverage(apartment, devicePoint, direction, deviceRange, deviceAngle)
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

// CalculateAreaCoverage вычисляет охватываемую площадь комнаты по заданной точке для устройства
func (r *Room) CalculateAreaCoverage(apartment *Apartment, devicePoint point.Point, deviceDirection point.Point, deviceRange float64, deviceAngle float64) (float64, error) {
	gridPoints, err := point.GenerateGridPoints(r.Area, 0.1)
	if err != nil {
		return 0, err
	}

	if len(gridPoints) == 0 {
		return 0, nil
	}

	if deviceAngle <= 0 {
		deviceAngle = baseDeviceAngle
	}

	if deviceRange <= 0 {
		deviceRange = math.MaxInt
	}

	visiblePoints := 0

	for _, p := range gridPoints {
		if r.IsPointVisibleOnDevice(apartment, p, devicePoint, deviceRange, deviceAngle, deviceDirection) {
			visiblePoints++
		}
	}

	return float64(visiblePoints) / float64(len(gridPoints)), nil
}

// IsPointVisibleOnDevice проверяет, видна ли точка на устройстве
func (r *Room) IsPointVisibleOnDevice(apartment *Apartment, p point.Point, devicePoint point.Point, deviceRange, deviceAngle float64, deviceDirection point.Point) bool {
	if point.CalculatePointsDistance(devicePoint, p) > deviceRange {
		return false
	}

	vectorDeviceToPoint := point.NewVector(devicePoint, p)

	if vectorDeviceToPoint.X != 0 || vectorDeviceToPoint.Y != 0 {
		vecLen := math.Sqrt(vectorDeviceToPoint.X*vectorDeviceToPoint.X + vectorDeviceToPoint.Y*vectorDeviceToPoint.Y)
		vectorDeviceToPoint.X /= vecLen
		vectorDeviceToPoint.Y /= vecLen

		dotPr := vectorDeviceToPoint.X*deviceDirection.X + vectorDeviceToPoint.Y*deviceDirection.Y
		angle := math.Acos(dotPr) * (180 / math.Pi)

		if angle > deviceAngle/2 {
			return false
		}
	}

	deviceOffset := r.GetDeviceOffset(apartment, devicePoint, deviceDirection)
	devicePoint = point.MovePointInDirection(devicePoint, deviceDirection, deviceOffset)

	return !apartment.IsWallBetweenPoints(p, devicePoint)
}

// GetDeviceOffset определяет сдвиг для устройства
// (чтобы после корректно использовать проверку стены между точками)
func (r *Room) GetDeviceOffset(apartment *Apartment, devicePoint point.Point, deviceDirection point.Point) float64 {
	minWallLen := math.MaxFloat64

	for _, wallID := range r.Walls {
		wall := apartment.wallsByID[wallID]
		minWallLen = min(minWallLen, point.CalculatePointsDistance(wall.Points[0], wall.Points[1]))
	}

	return minWallLen * 0.01
}
