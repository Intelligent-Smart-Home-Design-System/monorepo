package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/google/uuid"
)

type IlluminationSensorRule struct {
	track string
}

func NewIlluminationSensorRule() *IlluminationSensorRule {
	return &IlluminationSensorRule{
		track: "lighting",
	}
}

func (r *IlluminationSensorRule) Type() string {
	return "illumination_sensor"
}

func (r *IlluminationSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
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

		place, err := illuminationPoint(apartmentStruct, room)
		if err != nil {
			return err
		}

		deviceID := uuid.NewString()
		dev := device.NewDevice(deviceID, r.Type(), r.track)
		placement := device.NewPlacement(dev, roomID, place)
		apartmentLayout.Placements[roomID][dev.Type] = placement
	}

	return nil
}

func illuminationPoint(apartmentStruct *apartment.Apartment, room apartment.Room) (*point.Point, error) {
	if len(room.Area) == 0 {
		fallback := point.Point{X: 0, Y: 0}
		return &fallback, nil
	}

	roomCenter, err := room.GetCenter()
	if err != nil {
		fallback := room.Area[0]
		return &fallback, nil
	}

	roomWindows := getRoomWindows(apartmentStruct, room.ID)
	// Если в комнате есть окна, ставим датчик в ближайший угол от окна
	if len(roomWindows) > 0 && len(roomWindows[0].Points) > 0 {
		windowCenter := apartment.GetObjectCenter(roomWindows[0].Points)
		return cornerNearWindow(room.Area, windowCenter), nil
	}

	// Если окна нет, берем самый дальний угол от центра (подальше от лампочки)
	return farCornerFromCenter(room.Area, *roomCenter), nil
}

func getRoomWindows(apartmentStruct *apartment.Apartment, roomID string) []apartment.Window {
	windows := make([]apartment.Window, 0)

	for _, w := range apartmentStruct.Windows {
		for _, connectedRoomID := range w.Rooms {
			if connectedRoomID == roomID {
				windows = append(windows, w)
				break
			}
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
