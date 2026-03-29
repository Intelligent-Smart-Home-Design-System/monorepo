package entities

func (a *Apartment) GetRoomsByType(roomType string) []*Room {
	rooms := make([]*Room, 0)

	for _, room := range a.Rooms {
		if room.Name == roomType {
			rooms = append(rooms, room)
		}
	}

	return rooms
}
