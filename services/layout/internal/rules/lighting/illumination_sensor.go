package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
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

func (r *IlluminationSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	devicesRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	for _, room := range devicesRooms {
		roomID := room.ID

		place, err := illuminationPoint(apartmentStruct, room)
		if err != nil {
			return err
		}

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, place, nil)
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
	if len(roomWindows) > 0 && len(roomWindows[0].Points) > 0 {
		windowCenter := apartment.GetObjectCenter(roomWindows[0].Points)
		return cornerNearWindow(room.Area, windowCenter), nil
	}

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
