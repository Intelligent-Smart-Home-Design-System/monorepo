package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/geometry"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const track string = "climate"

const wattsPerM2 float64 = 100
const powerMargin float64 = 1.2
const wattToBTU float64 = 3.412

type AirConditioner struct{}

func NewAirConditionerRule() *AirConditioner {
	return &AirConditioner{}
}

func (r *AirConditioner) Type() string {
	return "air_conditioner"
}

func (r *AirConditioner) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	roomsSet := make(map[string]struct{})
	for _, name := range deviceRooms {
		roomsSet[name] = struct{}{}
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
		acWidthM = float64(acFilter.IndoorUnitLengthMM) / 1000.0
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
		midOffset := bestInterval.From + bestInterval.Length()/2
		wallSeg := point.Segment{From: wall.Points[0], To: wall.Points[1]}
		dir := wallSeg.Direction()
		p := point.MovePointInDirection(wall.Points[0], dir, midOffset)

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

func collectNoWindZones(furniture []*apartment.Furniture) []*apartment.Zone {
	zones := make([]*apartment.Zone, 0)
	for _, f := range furniture {
		if f.Category == apartment.Bed {
			zones = append(zones, apartment.NewZone(f.Points))
		}
	}
	return zones
}

func estimateRequiredCoolingWatts(areaM2 float64) float64 {
	return areaM2 * wattsPerM2 * powerMargin
}

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

func FindLongestInterval(
	walls []apartment.Wall,
	intervals map[string][]point.Interval,
	availableWallsID map[string]struct{},
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