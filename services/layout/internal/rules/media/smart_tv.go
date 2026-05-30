package media

import (
	"slices"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/geometry"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const defaultTVWidth = 1.5

type SmartTVRule struct {
	track string
}

func NewSmartTVRule() *SmartTVRule {
	return &SmartTVRule{
		track: "media",
	}
}

func (stv *SmartTVRule) Type() string {
	return "smart_tv"
}

// func (c *SmartTVRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
// 	rooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
// 	if err != nil {
// 		return err
// 	}

// 	roomsSet := make(map[string]struct{})
// 	for _, r := range rooms {
// 		roomsSet[r.Name] = struct{}{}
// 	}

// 	for _, zr := range zonedAp.ZonedRooms {
// 		if _, ok := roomsSet[zr.OrigRoom.Name]; ok {
// 			zr.TVZones = collectTVZones(zonedAp.OrigAp, zr)
// 		}
// 	}

// 	return nil
// }

func (stv *SmartTVRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := stv.Type()

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(stv.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.SmartTVFilter{
			Width: defaultTVWidth,
		}
	}
	smartTVFilters := configFilters.(*filters.SmartTVFilter)

	smartTVRooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, r := range smartTVRooms {
		roomsSet[r.Name] = struct{}{}
	}

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; !ok || deviceCnt >= maxCount {
			continue
		}

		bestPoint, maxWidth := findBestTVPoint(zonedAp.OrigAp, zr, smartTVFilters.Width)
		if bestPoint == nil {
			continue
		}

		deviceFilter := &filters.SmartTVFilter{
			Resolution: smartTVFilters.Resolution,
			Width: smartTVFilters.Width,
			RefreshRatehHZ: smartTVFilters.RefreshRatehHZ,
			MaxWidthM: maxWidth,
		}

		layout.AddDeviceToLayout(deviceType, stv.track, zr.OrigRoom.ID, bestPoint, nil, deviceFilter)
		deviceCnt++
	}

	return nil
}

func findBestTVPoint(ap *apartment.Apartment, zr *apartment.ZonedRoom, tvWidth float64) (*point.Point, float64) {
	room := zr.OrigRoom
	intervals := make(map[string][]point.Interval)

	for _, wallID := range room.Walls {
		if zr.ACAvailableWalls != nil {
			if _, ok := zr.ACAvailableWalls[wallID]; !ok {
				continue
			}
		}
	
		wall, err := ap.GetWallByID(wallID)
        if err != nil {
            continue
        }
	
        wallSegment := point.Segment{From: wall.Points[0], To: wall.Points[1]}
        tracker := geometry.NewWallIntervalTracker(wallSegment.Length())
		for _, windowID := range room.Windows {
            window, err := ap.GetWindowByID(windowID)
            if err != nil {
                continue
            }

            if isWindowOnWall(window, wall) {
                windowInterval, _ := geometry.ProjectPolygonToSegment(wallSegment, window.Points)
                if windowInterval != nil {
                    tracker.Block(*windowInterval)
                }
            }
        }
        
        for _, dID := range room.Doors {
            door, err := ap.GetDoorByID(dID)
            if err != nil {
                continue
            }

            if isDoorOnWall(door, wall) {
                doorInterval, _ := geometry.ProjectPolygonToSegment(wallSegment, door.Points)
                if doorInterval != nil {
                    tracker.Block(*doorInterval)
                }
            }
        }
        
        free := tracker.FreeIntervals(tvWidth)
        if len(free) > 0 {
            intervals[wall.ID] = free
        }
    }
	
	var bestPoint point.Point
	tvFurnitures := getTVFurnitureInRoom(ap, room)
	for _, furniture := range tvFurnitures {
		if furniture != nil {
			for wID, freeIntervals := range intervals {
				wall, _ := ap.GetWallByID(wID)
				
				for _, iv := range freeIntervals {
					if iv.Length() >= tvWidth {
						bestPoint = getPointOnWall(wall, iv)
						return &bestPoint, iv.Length()
					}
				}
			}
		}
	}

    var maxSize float64
    for wID, freeIntervals := range intervals {
        for _, iv := range freeIntervals {
            if iv.Length() > maxSize {
				wall, err := ap.GetWallByID(wID)
				if err != nil {
					break
				}

                maxSize = iv.Length()
                bestPoint = getPointOnWall(wall, iv)
            }
        }
    }

	return &bestPoint, maxSize
}

func isWindowOnWall(window *apartment.Window, wall *apartment.Wall) bool {
	windowCenter := point.GetObjectCenter(window.Points)

	return point.IsPointOnSegment(wall.Points[0], windowCenter, wall.Points[1])
}

func isDoorOnWall(door *apartment.Door, wall *apartment.Wall) bool {
	doorCenter := point.GetObjectCenter(door.Points)

	return point.IsPointOnSegment(wall.Points[0], doorCenter, wall.Points[1])
}

func getTVFurnitureInRoom(ap *apartment.Apartment, room *apartment.Room) []*apartment.Furniture {
	result := make([]*apartment.Furniture, 0)

	for _, fID := range room.Furniture {
		furniture, err := ap.GetFurnitureByID(fID)
		if err != nil {
			continue
		}

		switch furniture.Name {
		case apartment.Sofa, apartment.Bed:
			result = append(result, furniture)
		}
	}

	slices.SortFunc(result, func(f1, f2 *apartment.Furniture) int {
		if f1.Name == apartment.Sofa {
			return -1
		}

		return 1
	})

	return result
}

func getPointOnWall(wall *apartment.Wall, iv point.Interval) point.Point {
	wallSegment := point.Segment{From: wall.Points[0], To: wall.Points[1]}
	direction := wallSegment.Direction()

	return point.MovePointInDirection(wallSegment.From, direction, (iv.From + iv.To) / 2)
}
