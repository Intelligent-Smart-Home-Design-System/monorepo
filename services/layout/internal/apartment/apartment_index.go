package apartment

import "fmt"

// Index создает вспомогающую зависимость в квартире
func (a *Apartment) Index() {
	a.IndexRooms()
	a.IndexWalls()
	a.IndexFurniture()
	a.bindRooms()
}

// IndexRooms создает словарь зависимости названия комнаты
// со слайсом структур комнат с таким же названием
func (a *Apartment) IndexRooms() {
	a.roomsByName = make(map[string][]Room)

	for _, room := range a.Rooms {
		a.roomsByName[room.Name] = append(a.roomsByName[room.Name], room)
	}
}

// IndexWalls создает словарь зависимости индекса стены с его структурой
func (a *Apartment) IndexWalls() {
	a.wallsByID = make(map[string]*Wall)

	for i := range a.Walls {
		a.wallsByID[a.Walls[i].ID] = &a.Walls[i]
	}
}

func (a *Apartment) IndexFurniture() {
	a.furnitureByID = make(map[string]*Furniture)

	for i := range a.Furniture {
		a.furnitureByID[a.Furniture[i].ID] = &a.Furniture[i]
	}
}

func (a *Apartment) bindRooms() {
	for i := range a.Rooms {
		a.Rooms[i].apartment = a
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

func (a *Apartment) GetWallByID(id string) (*Wall, error) {
	wall, ok := a.wallsByID[id]
	if !ok {
		return nil, fmt.Errorf("wall with id %s not found", id)
	}
	return wall, nil
}

func (a *Apartment) GetFurnitureByID(id string) (*Furniture, error) {
	f, ok := a.furnitureByID[id]
	if !ok {
		return nil, fmt.Errorf("furniture with id %s not found", id)
	}
	return f, nil
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
