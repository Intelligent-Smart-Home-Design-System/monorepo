package apartment

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

// IsWallBetweenPoints проверяет, есть ли стены между двумя точками.
func (a *Apartment) IsWallBetweenPoints(A, B point.Point) bool {
	segAB := &point.Segment{From: A, To: B}

	for _, wall := range a.Walls {
		x1, y1, x2, y2 := wall.Points[0].X, wall.Points[0].Y, wall.Points[1].X, wall.Points[1].Y
		if A.IsInInterval(x1, y1, x2, y2) && B.IsInInterval(x1, y1, x2, y2) {
			continue
		}

		wallSeg := &point.Segment{From: wall.Points[0], To: wall.Points[1]}
		if point.IsSegmentsIntersect(segAB, wallSeg) {
			return true
		}
	}

	return false
}

func (a *Apartment) GetEntryDoors(room *Room) ([]*Door, error) {
	entryDoors := make([]*Door, 0)
	for _, dID := range room.Doors {
		door, ok := a.doorsByID[dID]
		if !ok {
			return nil, fmt.Errorf("Failed to find door with index %s", dID)
		}

		if len(door.Rooms) == 1 {
			entryDoors = append(entryDoors, door)
		}
	}

	return entryDoors, nil
}
