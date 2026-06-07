package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type DoorSensorRule struct {
	track string
}

func NewDoorSensorRule() *DoorSensorRule {
	return &DoorSensorRule{
		track: "security",
	}
}

func (ds *DoorSensorRule) Type() string {
	return "door_sensor"
}

func (ds *DoorSensorRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
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

func (ds *DoorSensorRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := ds.Type()

	err := ds.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(ds.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.DoorSensorFilter{}
	}
	doorSensorFilters := configFilters.(*filters.DoorSensorFilter)

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		if zr.EntryDoorZone == nil {
			continue
		}
	
		zoneCenter := point.GetObjectCenter(zr.EntryDoorZone.Points)

		if deviceCnt < maxCount {
			layout.AddDeviceToLayout(deviceType, ds.track, zr.OrigRoom.ID, &zoneCenter, nil, doorSensorFilters)
			deviceCnt++
		}
	}

	return nil
}
