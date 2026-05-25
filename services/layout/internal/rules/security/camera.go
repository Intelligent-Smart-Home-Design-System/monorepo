package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type CameraRule struct {
	track string
}

func NewCameraRule() *CameraRule {
	return &CameraRule{
		track: "security",
	}
}

func (c *CameraRule) Type() string {
	return "camera"
}

func (c *CameraRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	rooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, r := range rooms {
		roomsSet[r.Name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; ok && zr.OrigRoom.Name == apartment.RoomHall {
			zr.CameraZones = collectHighTrafficZones(zonedAp.OrigAp, zr.OrigRoom)
			
		}
	}

	return nil
}

func (c *CameraRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := c.Type()

	// err := c.Transform(zonedAp, deviceRooms)
	// if err != nil {
	// 	return err
	// }

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(c.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.CameraFilter{}
	}
	cameraFilters := configFilters.(*filters.CameraFilter)

	cameraRooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	deviceCnt := 0
	for _, room := range cameraRooms {
		roomID := room.ID

		maxDistance := room.CalculateMaxDistance()
		cameraFilters.Range = maxDistance

		cameraPoint, err := GetBestCameraPoint(zonedAp.OrigAp, room, cameraFilters.Range, cameraFilters.Angle)
		if err != nil {
			return err
		}

		if deviceCnt < maxCount {
			layout.AddDeviceToLayout(deviceType, c.track, roomID, cameraPoint, cameraFilters)
			deviceCnt++
		}
	}

	return nil
}

// GetBestCameraPoint возвращает лучшую по алгоритму точку в комнате для камеры.
// В прихожей камера ставится напротив входной двери.
// В остальных комнатах камера ставится в том месте, в котором охватывается наибольшая площадь комнаты
func GetBestCameraPoint(ap *apartment.Apartment, room *apartment.Room, deviceRange float64, deviceAngle float64) (*point.Point, error) {
	if room.Name == apartment.RoomHall {
		entryDoor := room.GetEntryDoor(ap)
		if entryDoor == nil {
			return nil, nil
		}

		return GetBestHallCameraPoint(room, entryDoor)
	}

	return room.GetMaxAreaDevicePoint(ap, deviceRange, deviceAngle)
}


// GetBestHallCameraPoint возвращает лучшую точку для камеры в прихожей (напротив входной двери)
func GetBestHallCameraPoint(room *apartment.Room, entryDoor *apartment.Door) (*point.Point, error) {
	doorCenter := point.GetObjectCenter(entryDoor.Points)

	bestPoint := room.Area[0]
	maxDist := point.CalculatePointsDistance(doorCenter, bestPoint)

	for _, p := range room.Area[1:] {
		dist := point.CalculatePointsDistance(doorCenter, p)
		if dist > maxDist {
			maxDist = dist
			bestPoint = p
		}
	}

	return &bestPoint, nil
}