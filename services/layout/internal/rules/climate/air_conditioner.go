package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/geometry"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const track string = "climate"

// wattsPerM2 — мощность охлаждения на 1 м² (Вт).
// https://www.dns-shop.ru/guide/17a8d3a3-1640-11e5-a679-00259074e77d/#rekomenduemaa-plosad-pomesenia
// Для охлаждения одного квадратного метра помещения необходимо около 100 Вт.
const wattsPerM2 float64 = 100

// powerMargin — запас мощности для более эффективной работы и снижения нагрузок (20%).
const powerMargin float64 = 1.2

// wattToBTU — коэффициент перевода Вт в BTU/ч.
const wattToBTU float64 = 3.412

type AirConditioner struct{}

func NewAirConditionerRule() *AirConditioner {
	return &AirConditioner{}
}

func (r *AirConditioner) Type() string {
	return "air_conditioner"
}

// Transform строит ZonedApartment из Apartment, обогащая комнаты зонами для кондиционера.
func (r *AirConditioner) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	rooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, rm := range rooms {
		roomsSet[rm.Name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; ok {
			zr.NoWindZones = collectNoWindZones(zr.GetFurniture())
		}
	}

	return nil
}

func (ac *AirConditioner) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	err := ac.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(track, levelNum, "air_conditioner")
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.AirConditionerFilter{}
	}
	acFilter := configFilters.(*filters.AirConditionerFilter)

	var acWidthM float64
	if acFilter != nil && acFilter.IndoorUnitLengthMM > 0 {
		acWidthM = acFilter.IndoorUnitLengthMM / 1000.0
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok {
			continue
		}

		intervals := FindOkWindWallIntervals(zr.OrigRoom, zr.NoWindZones, acWidthM)
		bestWallID, bestInterval := FindLongestInterval(zonedAp.OrigAp.Walls, intervals, zr.ACAvailableWalls)

		if bestInterval == nil {
			continue
		}

		wall, err := zonedAp.OrigAp.GetWallByID(bestWallID)
		if err != nil {
			continue
		}

		// Вычисляем точку размещения: середина лучшего интервала на стене
		midOffset := bestInterval.From + bestInterval.Length()/2
		wallSeg := point.Segment{From: wall.Points[0], To: wall.Points[1]}
		dir := wallSeg.Direction()
		p := point.MovePointInDirection(wall.Points[0], dir, midOffset)

		// Рассчитываем необходимую мощность: 100 Вт/м² * площадь * 1.2 (запас 20%)
		requiredWatts := estimateRequiredCoolingWatts(zr.OrigRoom.AreaM2)
		requiredBTU := requiredWatts * wattToBTU

		var deviceFilter filters.DeviceFilter
		if acFilter != nil {
			deviceFilter = &filters.AirConditionerFilter{
				NoiseLevelDB:       acFilter.NoiseLevelDB,
				MaxNoiseLevelDB:    acFilter.MaxNoiseLevelDB,
				CoolingPowerBTU:    requiredBTU,
				CoolingPowerWatts:  requiredWatts,
				IndoorUnitLengthMM: acFilter.IndoorUnitLengthMM,
				RecommendedAreaM2:  zr.OrigRoom.AreaM2 * powerMargin,
			}
		}

		layout.AddDeviceToLayout(ac.Type(), track, zr.OrigRoom.ID, &p, nil, deviceFilter)
	}

	return nil
}

// collectNoWindZones собирает непродуваемые зоны из мебели (кровати).
func collectNoWindZones(furniture []*apartment.Furniture) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)
	for _, f := range furniture {
		if f.Name == apartment.Bed {
			zones = append(zones, apartment.NewZone(f.Points))
		}
	}
	return zones
}

// estimateRequiredCoolingWatts рассчитывает необходимую мощность охлаждения (Вт) для комнаты.
// Формула: 100 Вт/м² * площадь * 1.2 (запас 20%).
func estimateRequiredCoolingWatts(areaM2 float64) float64 {
	return areaM2 * wattsPerM2 * powerMargin
}

// FindOkWindWallIntervals находит свободные интервалы на стенах комнаты,
// где можно разместить кондиционер без попадания потока на непродуваемые зоны.
func FindOkWindWallIntervals(room *apartment.Room, forbiddenZones []*apartment.Zone, acWidth float64) map[string][]point.Interval {
	result := make(map[string][]point.Interval)
	walls := room.GetWalls()

	for i, wall := range walls {
		wallSeg := point.Segment{From: wall.Points[0], To: wall.Points[1]}
		tracker := geometry.NewWallIntervalTracker(wallSeg.Length())

		for _, zone := range forbiddenZones {
			proj, zonePoints := geometry.ProjectPolygonToSegment(wallSeg, zone.Points)
			if proj == nil {
				continue
			}
			dir := wallSeg.Direction()
			startPoint := wall.Points[0]
			pFrom := point.MovePointInDirection(startPoint, dir, proj.From)
			pTo := point.MovePointInDirection(startPoint, dir, proj.To)

			zoneRect := []point.Point{zonePoints.From, zonePoints.To, pTo, pFrom}

			tracker.Block(*proj)

			for j, block := range walls {
				if i == j {
					continue
				}
				blockSeg := point.Segment{From: block.Points[0], To: block.Points[1]}
				if !geometry.IsSegmentIntersectPolygon(zoneRect, blockSeg) {
					continue
				}

				blockProj, _ := geometry.ProjectPolygonToSegment(wallSeg, []point.Point{block.Points[0], block.Points[1]})
				if blockProj == nil {
					continue
				}

				tracker.Protect(*blockProj)
			}
		}

		free := tracker.FreeIntervals(acWidth)
		if len(free) > 0 {
			result[wall.ID] = free
		}
	}

	return result
}

// FindLongestInterval находит стену и интервал с максимальной длиной.
func FindLongestInterval(
	walls []apartment.Wall,
	intervals map[string][]point.Interval,
	availableWallsID map[string]struct{}, // nil = все доступны
) (bestWallID string, bestInterval *point.Interval) {
	for wallID, ivList := range intervals {
		if availableWallsID != nil {
			if _, ok := availableWallsID[wallID]; !ok {
				continue
			}
		}

		if len(ivList) == 0 {
			continue
		}

		candidate := ivList[0]

		if bestInterval == nil || candidate.Length() > bestInterval.Length() {
			bestWallID = wallID
			bestInterval = &candidate
		}
	}

	return
}
