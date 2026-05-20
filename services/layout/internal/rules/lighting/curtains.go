package lighting

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"

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
func (r *CurtainsRule) Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error {
	for _, w := range apartmentStruct.Windows {
		if len(w.Rooms) == 0 {
			continue
		}
		if len(w.Points) == 0 {
			continue
		}

		roomID := w.Rooms[0]
		windowCenter := apartment.GetObjectCenter(w.Points)

		layout.AddDeviceToLayout(r.Type(), r.track, roomID, &windowCenter, nil)
	}

	return nil
}
