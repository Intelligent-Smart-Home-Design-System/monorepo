package apartment

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"

type Apartment struct {
	ID     string   `json:"id"`
	Tracks []string `json:"tracks"`
	Rooms  []*Room  `json:"rooms"`
}

type ApartmentResult struct {
	Placements map[string]map[string]*device.Placement // roomID -> deviceType -> devicePlacement
	// То есть по roomID получаем мапу между
	// типом устройства и его расстановкой

	// в дальнейшем необходимо будет хранить доп поля в этой структуре (для других модулей)
}

func NewApartmentResult() *ApartmentResult {
	return &ApartmentResult{Placements: make(map[string]map[string]*device.Placement)}
}

type Room struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	WetPoints []*device.Point `json:"wet_points"`
}

func (a *Apartment) GetRoomsByType(roomType string) []*Room {
	rooms := make([]*Room, 0)

	for _, room := range a.Rooms {
		if room.Name == roomType {
			rooms = append(rooms, room)
		}
	}

	return rooms
}
