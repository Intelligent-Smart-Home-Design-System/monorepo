package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
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

			if deviceCnt < maxCount {
				layout.AddDeviceToLayout(deviceType, sd.track, zr.OrigRoom.ID, &zoneCenter, smartDoorBellFilters)
				deviceCnt++
			}
		}
	}

	return nil
}
