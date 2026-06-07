package security

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type SmartDoorBellRule struct {
	track string
}

func NewSmartDoorBellRule() *SmartDoorBellRule {
	return &SmartDoorBellRule{
		track: "security",
	}
}

func (sd *SmartDoorBellRule) Type() string {
	return "smart_doorbell"
}

func (sd *SmartDoorBellRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
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
			zr.EntryDoorZone = collectEntryDoorZone(zonedAp.OrigAp, zr.OrigRoom)
		}
	}

	return nil
}

func (sd *SmartDoorBellRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := sd.Type()
	err := sd.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}
	
	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(sd.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.SmartDoorBellFilter{}
	}
	smartDoorBellFilters := configFilters.(*filters.SmartDoorBellFilter)

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if zr.EntryDoorZone != nil && zr.OrigRoom.Name == apartment.RoomHall {
			zoneCenter := zr.EntryDoorZone.Points[0]

			// Записать направление
			_, err := findDoorbellDirection(zr)
			if err != nil {
				continue
			}

			if deviceCnt < maxCount {
				layout.AddDeviceToLayout(deviceType, sd.track, zr.OrigRoom.ID, &zoneCenter, nil, smartDoorBellFilters)
				deviceCnt++
			}
		}
	}

	return nil
}

func collectEntryDoorZone(ap *apartment.Apartment, room *apartment.Room) *apartment.Zone {
	entryDoor := room.GetEntryDoor(ap)
	if entryDoor == nil {
		return nil
	}

	return apartment.NewZone(entryDoor.Points)
}

func findDoorbellDirection(zr *apartment.ZonedRoom) (*point.Point, error) {
	if zr.EntryDoorZone == nil {
		return nil, fmt.Errorf("failed to find entry door zone")
	}

	return zr.OrigRoom.GetOppositeDirectionToRoom(&point.Segment{
		From: zr.EntryDoorZone.Points[0],
		To: zr.EntryDoorZone.Points[1],
	}), nil
}
