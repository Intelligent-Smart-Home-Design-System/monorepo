package apartment

import (
	"math"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const (
	baseDeviceAngle    = 100
	doorZoneDepth      = 1.5
	degreePerDirection = 30
)

// IsPointInRoom проверяет, находится ли точка в комнате
func (r *Room) IsPointInRoom(targetPoint point.Point) bool {
	return point.IsPointInPolygon(targetPoint, r.Area)
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

func (r *Room) CreateObjectZone(objectPoints []point.Point, objectWidth float64) *Zone {
	roomCenter := r.Center
	if roomCenter == nil {
		roomCenter = point.GetCenter(r.Area)
	}

	var points []point.Point

	objectCenter := point.GetObjectCenter(objectPoints)
	halfWidth := objectWidth / 2

	dx := roomCenter.X - objectCenter.X
	dy := roomCenter.Y - objectCenter.Y

	if objectPoints[0].X == objectPoints[1].X {
		if dx > 0 {
			points = []point.Point{
				{X: objectCenter.X, Y: objectCenter.Y - halfWidth},
				{X: objectCenter.X, Y: objectCenter.Y + halfWidth},
				{X: objectCenter.X + doorZoneDepth, Y: objectCenter.Y + halfWidth},
				{X: objectCenter.X + doorZoneDepth, Y: objectCenter.Y - halfWidth},
			}
		} else {
			points = []point.Point{
				{X: objectCenter.X, Y: objectCenter.Y - halfWidth},
				{X: objectCenter.X, Y: objectCenter.Y + halfWidth},
				{X: objectCenter.X - doorZoneDepth, Y: objectCenter.Y + halfWidth},
				{X: objectCenter.X - doorZoneDepth, Y: objectCenter.Y - halfWidth},
			}
		}
	} else {
		if dy > 0 {
			points = []point.Point{
				{X: objectCenter.X - halfWidth, Y: objectCenter.Y},
				{X: objectCenter.X - halfWidth, Y: objectCenter.Y + doorZoneDepth},
				{X: objectCenter.X + halfWidth, Y: objectCenter.Y + doorZoneDepth},
				{X: objectCenter.X + halfWidth, Y: objectCenter.Y},
			}
		} else {
			points = []point.Point{
				{X: objectCenter.X - halfWidth, Y: objectCenter.Y},
				{X: objectCenter.X - halfWidth, Y: objectCenter.Y - doorZoneDepth},
				{X: objectCenter.X + halfWidth, Y: objectCenter.Y - doorZoneDepth},
				{X: objectCenter.X + halfWidth, Y: objectCenter.Y},
			}
		}
	}

	return NewZone(points)
}

func (r *Room) GetTheOppositePoint(p point.Point) (point.Point, float64) {
	bestPoint := r.Area[0]
	maxDist := point.CalculatePointsDistance(p, bestPoint)

	for _, corner := range r.Area[1:] {
		dist := point.CalculatePointsDistance(p, corner)
		if dist > maxDist {
			maxDist = dist
			bestPoint = corner
		}
	}

	return bestPoint, maxDist
}

func FindBestDirectionForDevicePoint(ap *Apartment, zr *ZonedRoom, zones []*Zone, devicePoint point.Point, deviceRange, deviceAngle float64) (point.Point, float64) {
	var bestDirection point.Point
	maxCoverage := 0.0

	for i := 0; i < 360; i += degreePerDirection {
		angle := float64(i) * math.Pi / 180
		direction := point.Point{
			X: math.Cos(angle),
			Y: math.Sin(angle),
		}

		coverage := calculateDeviceZoneCoverage(ap, zr, zones, devicePoint, deviceRange, deviceAngle, direction)
		if maxCoverage < coverage {
			maxCoverage = coverage
			bestDirection = direction
		}
	}

	return bestDirection, maxCoverage
}

func calculateDeviceZoneCoverage(ap *Apartment, zr *ZonedRoom, zones []*Zone, devicePoint point.Point, deviceRange, deviceAngle float64, direction point.Point) float64 {
	if len(zones) == 0 {
		return 1
	}

	coveredZones := 0
	for _, zone := range zones {
		zoneCenter := point.GetCenter(zone.Points)
		if zr.OrigRoom.IsPointVisibleOnDevice(ap, *zoneCenter, devicePoint, deviceRange, deviceAngle, direction) {
			coveredZones++
		}

		for _, p := range zone.Points {
			if zr.OrigRoom.IsPointVisibleOnDevice(ap, p, devicePoint, deviceRange, deviceAngle, direction) {
				coveredZones++
			}
		}
	}

	return float64(coveredZones) / float64(len(zones) * 5)
}

func (r *Room) GetOppositeDirectionToRoom(s *point.Segment) *point.Point {
	dir1, dir2 := s.NormalVectors()

	roomCenter := r.Center
	if roomCenter == nil {
		roomCenter = point.GetCenter(r.Area)
	}
	
	dx := roomCenter.X - (s.From.X + s.From.X) / 2
	dy := roomCenter.Y - (s.From.Y + s.From.Y) / 2

	if math.Abs(dx) > math.Abs(dy) {
		if dx > 0 {
			return dir1
		} else {
			return dir2
		}
	}

	if dy > 0 {
		return dir1
	}
	return dir2
}
