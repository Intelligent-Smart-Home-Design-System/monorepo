package security

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
	"github.com/google/uuid"
)

type WaterLeakSensorRule struct {
	track string
}

func NewWaterLeakRule() *WaterLeakSensorRule {
	return &WaterLeakSensorRule{
		track: "security",
	}
}

func (wl *WaterLeakSensorRule) GetType() string {
	return "water_leak_sensor"
}

func (wl *WaterLeakSensorRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error {
	wetRooms, err := apartmentStruct.GetRoomsByNames(deviceRooms)
	if err != nil {
		return err
	}

	// TODO: улучшить, когда появятся мокрые точки на плане

	for _, room := range wetRooms {
		roomID := room.ID

		_, ok := apartmentLayout.Placements[roomID]
		if !ok {
			apartmentLayout.Placements[roomID] = make(map[string]*device.Placement)
		}

		roomCenter, err := room.GetCenter()
		if err != nil {
			return err
		}

		deviceID := uuid.NewString() // в будущем все ID будут прописаны в конфигах и будут браться оттуда
		newDevice := device.NewDevice(deviceID, "water_leak_sensor", "security")
		placement := device.NewPlacement(newDevice, roomID, *roomCenter)

		apartmentLayout.Placements[roomID][newDevice.Type] = placement
	}

	return nil
}
