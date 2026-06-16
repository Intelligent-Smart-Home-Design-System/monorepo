package apartment

import (
	"fmt"
	"strings"
)

// Index создает вспомогающую зависимость в квартире
func (a *Apartment) Index() {
	a.IndexRooms()
	a.IndexWalls()
	a.IndexWindows()
	a.IndexDoors()
	a.IndexFurniture()
	// a.IndexPlumbing()
	// a.IndexAppliances()
	a.bindRooms()
}

// IndexRooms создает словарь зависимости названия комнаты
// со слайсом структур комнат с таким же названием
func (a *Apartment) IndexRooms() {
	a.roomsByName = make(map[string][]*Room)

	for i := range a.Rooms {
		a.Rooms[i].Name = strings.ToLower(a.Rooms[i].Name)

		name := a.Rooms[i].Name
		a.roomsByName[name] = append(a.roomsByName[name], &a.Rooms[i])
		switch name {
		case "hall":
			a.roomsByName["hallway"] = append(a.roomsByName["hallway"], &a.Rooms[i])
		case "hallway":
			a.roomsByName["hall"] = append(a.roomsByName["hall"], &a.Rooms[i])
		case "livingroom":
			a.roomsByName["living"] = append(a.roomsByName["living"], &a.Rooms[i])
		case "living":
			a.roomsByName["livingroom"] = append(a.roomsByName["livingroom"], &a.Rooms[i])
		}
	}
}

// IndexWalls создает словарь зависимости индекса стены с его структурой
func (a *Apartment) IndexWalls() {
	a.wallsByID = make(map[string]*Wall)

	for i := range a.Walls {
		a.wallsByID[a.Walls[i].ID] = &a.Walls[i]
	}
}

// IndexWindows создает словарь зависимости индекса стены с его структурой
func (a *Apartment) IndexWindows() {
	a.windowByID = make(map[string]*Window)

	for i := range a.Windows {
		a.windowByID[a.Windows[i].ID] = &a.Windows[i]
	}
}

// IndexDoors создает словарь зависимости индекса двери с его структурой
func (a *Apartment) IndexDoors() {
	a.doorsByID = make(map[string]*Door)

	for i := range a.Doors {
		a.doorsByID[a.Doors[i].ID] = &a.Doors[i]
	}
}

// IndexFurniture создает словарь зависимости ID мебели с его структурой
func (a *Apartment) IndexFurniture() {
	a.furnitureByID = make(map[string]*Furniture)

	for i := range a.Furniture {
		a.Furniture[i].Category = strings.ToLower(a.Furniture[i].Category)
		a.furnitureByID[a.Furniture[i].ID] = &a.Furniture[i]
	}
}

// // IndexPlumbing создает словарь зависимости ID сантехники с его структурой
// func (a *Apartment) IndexPlumbing() {
// 	a.plumbingByID = make(map[string]*Plumbing)
// 	if a.Plumbing == nil {
// 		return
// 	}

// 	for i := range a.Plumbing {
// 		a.plumbingByID[a.Plumbing[i].ID] = &a.Plumbing[i]
// 	}
// }

// // IndexAppliances создает словарь зависимости ID бытовой техники с его структурой
// func (a *Apartment) IndexAppliances() {
// 	a.appliancesByID = make(map[string]*Appliances)
// 	if a.Appliances == nil {
// 		return
// 	}

// 	for i := range a.Appliances {
// 		a.appliancesByID[a.Appliances[i].ID] = &a.Appliances[i]
// 	}
// }

// bindRooms устанавливает обратную ссылку на квартиру для каждой комнаты
func (a *Apartment) bindRooms() {
	for i := range a.Rooms {
		a.Rooms[i].apartment = a
	}
}

// GetRoomsByNames возвращает все комнаты, имеющие названия из входного слайса.
func (a *Apartment) GetRoomsByNames(roomNames []string) ([]*Room, error) {
	roomsRes := make([]*Room, 0)

	for _, roomName := range roomNames {
		rooms, _ := a.roomsByName[roomName]
		roomsRes = append(roomsRes, rooms...)
	}

	return roomsRes, nil
}

// GetWallByID возвращает стену по ID
func (a *Apartment) GetWallByID(id string) (*Wall, error) {
	wall, ok := a.wallsByID[id]
	if !ok {
		return nil, fmt.Errorf("wall with id %s not found", id)
	}
	return wall, nil
}

// GetWindowByID возвращает стену по ID
func (a *Apartment) GetWindowByID(id string) (*Window, error) {
	window, ok := a.windowByID[id]
	if !ok {
		return nil, fmt.Errorf("window with id %s not found", id)
	}
	return window, nil
}

// GetDoorByID возвращает стену по ID
func (a *Apartment) GetDoorByID(id string) (*Door, error) {
	door, ok := a.doorsByID[id]
	if !ok {
		return nil, fmt.Errorf("door with id %s not found", id)
	}
	return door, nil
}

// GetFurnitureByID возвращает мебель по ID
func (a *Apartment) GetFurnitureByID(id string) (*Furniture, error) {
	f, ok := a.furnitureByID[id]
	if !ok {
		return nil, fmt.Errorf("furniture with id %s not found", id)
	}
	return f, nil
}

// // GetPlumbingByID возвращает сантехнику по ID
// func (a *Apartment) GetPlumbingByID(id string) (*Plumbing, error) {
// 	p, ok := a.plumbingByID[id]
// 	if !ok {
// 		return nil, fmt.Errorf("plumbing with id %s not found", id)
// 	}
// 	return p, nil
// }

// // GetAppliancesByID возвращает бытовую технику по ID
// func (a *Apartment) GetAppliancesByID(id string) (*Appliances, error) {
// 	app, ok := a.appliancesByID[id]
// 	if !ok {
// 		return nil, fmt.Errorf("appliance with id %s not found", id)
// 	}
// 	return app, nil
// }

// GetFrontDoor возвращает входную дверь в квартиру
func (a *Apartment) GetFrontDoor() *Door {
	for _, door := range a.Doors {
		if len(door.Rooms) == 1 {
			return &door
		}
	}

	return nil
}
