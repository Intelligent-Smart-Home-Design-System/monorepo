package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type SmartLockRule struct {
	track string
}

func NewSmartLockRule() *SmartLockRule {
	return &SmartLockRule{
		track: "security",
	}
}

func (sl *SmartLockRule) Type() string {
	return "smart_lock"
}

func (sl *SmartLockRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
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

func (sl *SmartLockRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := sl.Type()
	
	err := sl.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}
	
	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(sl.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.SmartLockFilter{}
	}
	smartLockFilters := configFilters.(*filters.SmartLockFilter)

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if zr.EntryDoorZone != nil && zr.OrigRoom.Name == apartment.RoomHall {
			zoneCenter := point.GetObjectCenter(zr.EntryDoorZone.Points)

			if deviceCnt < maxCount {
				layout.AddDeviceToLayout(deviceType, sl.track, zr.OrigRoom.ID, &zoneCenter, smartLockFilters)
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
