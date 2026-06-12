package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

const (
	MetersCoverage = 30
	CoeffDB        = 1.2
)

type SmartSirenRule struct {
	track string
}

func NewSmartSirenRule() *SmartSirenRule {
	return &SmartSirenRule{
		track: "security",
	}
}

func (ss *SmartSirenRule) Type() string {
	return "smart_siren"
}

func (ss *SmartSirenRule) Transform(zonedAp *apartment.ZonedApartment, deviceRooms []string) error {
	rooms, err := zonedAp.OrigAp.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	roomsSet := make(map[string]struct{})
	for _, r := range rooms {
		roomsSet[r.Name] = struct{}{}
	}

	for _, zr := range zonedAp.ZonedRooms {
		if _, ok := roomsSet[zr.OrigRoom.Name]; ok {
			switch zr.OrigRoom.Name {
			case apartment.RoomHall, apartment.RoomPassage:
				zr.SirenZones = collectSirenZones(zr.OrigRoom)
			}
		}
	}

	return nil
}

func (ss *SmartSirenRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	deviceType := ss.Type()

	err := ss.Transform(zonedAp, deviceRooms)
	if err != nil {
		return err
	}

	tracksConfig := configs.GetGlobalTracksConfig()
	configFilters, err := tracksConfig.GetDeviceFilter(ss.track, levelNum, deviceType)
	if err != nil {
		return err
	}

	if configFilters == nil {
		configFilters = &filters.SmartSirenFilter{}
	}
	smartSirenFilters := configFilters.(*filters.SmartSirenFilter)

	deviceCnt := 0
	for _, zr := range zonedAp.ZonedRooms {
		for _, sirenZone := range zr.SirenZones {
			zoneCenter := point.GetCenter(sirenZone.Points)

			if deviceCnt < maxCount {
				if zr.OrigRoom.AreaM2 >= MetersCoverage {
					smartSirenFilters.VolumeDB *= CoeffDB
				}

				layout.AddDeviceToLayout(deviceType, ss.track, zr.OrigRoom.ID, zoneCenter, smartSirenFilters)
				deviceCnt++
			}
		}
	}

	return nil
}

func collectSirenZones(room *apartment.Room) []*apartment.Zone {
	return []*apartment.Zone{apartment.NewZone(room.Area)}
}
