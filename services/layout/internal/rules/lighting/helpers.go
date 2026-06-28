package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

func getRoomWindows(apartmentStruct *apartment.Apartment, roomName string) []apartment.Window {
	windows := make([]apartment.Window, 0)

	for _, w := range apartmentStruct.Windows {
		if w.Room == roomName {
			windows = append(windows, w)
		}
	}

	return windows
}

func cornerNearWindow(corners []point.Point, target point.Point) *point.Point {
	best := corners[0]
	bestDist := point.CalculatePointsDistance(best, target)

	for _, c := range corners[1:] {
		d := point.CalculatePointsDistance(c, target)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}

	return &best
}

func farCornerFromCenter(corners []point.Point, target point.Point) *point.Point {
	best := corners[0]
	bestDist := point.CalculatePointsDistance(best, target)

	for _, c := range corners[1:] {
		d := point.CalculatePointsDistance(c, target)
		if d > bestDist {
			bestDist = d
			best = c
		}
	}

	return &best
}

func corridorEndPoints(room apartment.Room) (*point.Point, *point.Point, error) {
	if len(room.Area) < 2 {
		center := point.GetCenter(room.Area)
		if center == nil {
			fallback := point.Point{X: 0, Y: 0}
			return &fallback, &fallback, nil
		}
		return center, center, nil
	}

	iBest, jBest := 0, 1
	maxDist := point.CalculatePointsDistance(room.Area[0], room.Area[1])

	for i := 0; i < len(room.Area); i++ {
		for j := i + 1; j < len(room.Area); j++ {
			d := point.CalculatePointsDistance(room.Area[i], room.Area[j])
			if d > maxDist {
				maxDist = d
				iBest, jBest = i, j
			}
		}
	}

	p1 := room.Area[iBest]
	p2 := room.Area[jBest]
	return &p1, &p2, nil
}

func getRoomDoors(apartmentStruct *apartment.Apartment, roomID string) []apartment.Door {
	doors := make([]apartment.Door, 0)

	for _, d := range apartmentStruct.Doors {
		for _, connectedRoomID := range d.Rooms {
			if connectedRoomID == roomID {
				doors = append(doors, d)
				break
			}
		}
	}

	return doors
}

func cornerNearDoor(apartmentStruct *apartment.Apartment, room apartment.Room) (*point.Point, error) {
	center := point.GetCenter(room.Area)
	if center == nil {
		fallback := point.Point{X: 0, Y: 0}
		center = &fallback
	}

	if len(room.Area) == 0 {
		return center, nil
	}

	roomDoors := getRoomDoors(apartmentStruct, room.ID)
	if len(roomDoors) == 0 {
		return center, nil
	}

	doorCenter := point.GetObjectCenter(roomDoors[0].Points)
	best := room.Area[0]
	minDist := point.CalculatePointsDistance(best, doorCenter)

	for _, p := range room.Area[1:] {
		d := point.CalculatePointsDistance(p, doorCenter)
		if d < minDist {
			minDist = d
			best = p
		}
	}

	return &best, nil
}
