package lighting

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type CurtainsRule struct {
	track string
}

func NewCurtainsRule() *CurtainsRule {
	return &CurtainsRule{
		track: "lighting",
	}
}

func (r *CurtainsRule) Type() string {
	return "curtains"
}

// работаем напрямую с окнами, 1 окно = 1 устройство curtains
func (r *CurtainsRule) Apply(zonedAp *apartment.ZonedApartment, levelNum string, deviceRooms []string, maxCount int, layout *apartment.Layout) error {
	apartmentStruct := zonedAp.OrigAp
	for _, w := range apartmentStruct.Windows {
		if len(w.Rooms) == 0 {
			continue
		}
		if len(w.Points) == 0 {
			continue
		}

		roomID := w.Rooms[0]
		windowCenter := point.GetObjectCenter(w.Points)

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, &windowCenter, nil, nil)
	}

	return nil
}
