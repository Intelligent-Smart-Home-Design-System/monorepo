package climate

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

const climateDeviceOffset = 400

func findFirstWall(zr *apartment.ZonedRoom) *apartment.Wall {
	if zr == nil {
		return nil
	}

	for _, wall := range zr.GetWalls() {
		if len(wall.Points) >= 2 {
			return wall
		}
	}

	return nil
}
