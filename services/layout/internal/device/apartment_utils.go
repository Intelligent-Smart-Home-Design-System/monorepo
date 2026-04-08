package entities

// MakeDependency создает вспомогающую зависимость в квартире
func (a *Apartment) MakeDependency() {
	a.MakeRoomDependency()
	a.MakeWallDependency()
}

// MakeRoomDependency создает словарь зависимости названия комнаты
// со слайсом структур комнат с таким же названием
func (a *Apartment) MakeRoomDependency() {
	a.roomsByName = make(map[string][]Room)

	for _, room := range a.Rooms {
		a.roomsByName[room.Name] = append(a.roomsByName[room.Name], room)
	}
}

// MakeWallDependency создает словарь зависимости индекса стены с его структурой
func (a *Apartment) MakeWallDependency() {
	a.wallsByID = make(map[string]Wall)

	for _, wall := range a.Walls {
		a.wallsByID[wall.ID] = wall
	}
}

// GetRoomsByNames возвращает все комнаты, имеющие названия из входного слайса.
func (a *Apartment) GetRoomsByNames(roomNames []string) ([]Room, error) {
	roomsRes := make([]Room, 0)

	for _, roomName := range roomNames {
		rooms, _ := a.roomsByName[roomName]
		roomsRes = append(roomsRes, rooms...)
	}

	return roomsRes, nil
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
