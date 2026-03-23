package entities

import (
	"slices"
)

// GetRoomsByType возвращает все комнаты, подходящие под паттерн.
// Паттерном может быть название комнаты или ее характеристика
func (a *Apartment) GetRoomsByType(roomType string) []Room {
	roomsRes := make([]Room, 0)

	for _, room := range a.Rooms {
		switch roomType {
		case "wet":
			if room.IsWet() {
				roomsRes = append(roomsRes, room)
			}
		case "kitchen":
			if room.Name == "kitchen" {
				roomsRes = append(roomsRes, room)
			}
		case "hall":
			if room.Name == "hall" {
				roomsRes = append(roomsRes, room)
			}
		case "living":
			if room.Name == "living" {
				roomsRes = append(roomsRes, room)
			}
		}
	}

	return roomsRes
}

// IsWet проверяет, может ли в комнате быть протечка.
// То есть (грубо говоря) является ли комната мокрой
func (r *Room) IsWet() bool {
	wetRooms := []string{"kitchen", "bathroom", "toilet"}

	return slices.Contains(wetRooms, r.Name)
}

// GetFrontDoor возвращает входную дверь в квартиру
func (a *Apartment) GetFrontDoor() *Door {
	for _, door := range a.Doors {
		if len(door.Rooms) == 1 {
			return &door
		}
	}

	return nil
}
