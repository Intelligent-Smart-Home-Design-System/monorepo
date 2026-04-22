package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/google/uuid"
)

type MotionSensorRule struct {
	track string
}

func NewMotionSensorRule() *MotionSensorRule {
	return &MotionSensorRule{
		track: "lighting",
	}
}

func (r *MotionSensorRule) Type() string {
	return "motion_sensor"
}

func (r *MotionSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	devicesRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID
		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
		}

		// Для коридора 2 датчика на концах
		if room.Name == apartment.RoomPassage {
			p1, p2, err := corridorEndPoints(room)
			if err != nil {
				return err
			}

			id1 := uuid.NewString()
			dev1 := device.NewDevice(id1, r.Type(), r.track)
			apartmentLayout.Placements[roomID][r.Type()] = device.NewPlacement(dev1, roomID, p1)

			id2 := uuid.NewString()
			dev2 := device.NewDevice(id2, r.Type(), r.track)
			apartmentLayout.Placements[roomID][r.Type()+"_2"] = device.NewPlacement(dev2, roomID, p2)

			continue
		}

		// В остальных комнатах по 1 датчику в углу рядом с дверью
		sensorPoint, err := cornerNearDoor(apartmentStruct, room)
		if err != nil {
			return err
		}

		id := uuid.NewString()
		dev := device.NewDevice(id, r.Type(), r.track)
		apartmentLayout.Placements[roomID][r.Type()] =
			device.NewPlacement(dev, roomID, sensorPoint)
	}

	return nil
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
	center, err := room.GetCenter()
	if err != nil {
		fallback := point.Point{X: 0, Y: 0}
		return &fallback, nil
	}

	if len(room.Area) == 0 {
		return center, nil
	}

	roomDoors := getRoomDoors(apartmentStruct, room.ID)
	if len(roomDoors) == 0 {
		return center, nil
	}

	doorCenter := apartment.GetObjectCenter(roomDoors[0].Points)
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

func corridorEndPoints(room apartment.Room) (*point.Point, *point.Point, error) {
	if len(room.Area) < 2 {
		center, err := room.GetCenter()
		if err != nil {
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
