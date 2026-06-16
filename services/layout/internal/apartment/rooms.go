package apartment

const (
	RoomLiving   = "livingroom"
	RoomBedroom  = "bedroom"
	RoomKitchen  = "kitchen"
	RoomPassage  = "passage"
	RoomBathroom = "bathroom"
	RoomCabinet  = "cabinet"
	RoomHall     = "hall"
)

// GetFurniture возвращает объекты мебели комнаты, разрешая ID через индекс квартиры.
func (r *Room) GetFurniture() []*Furniture {
	if r.apartment == nil {
		return nil
	}

	result := make([]*Furniture, 0, len(r.Furniture))
	for _, fID := range r.Furniture {
		f, err := r.apartment.GetFurnitureByID(fID)
		if err != nil {
			continue
		}
		result = append(result, f)
	}
	return result
}

// // GetPlumbing возвращает объекты сантехники комнаты, разрешая ID через индекс квартиры.
// func (r *Room) GetPlumbing() []*Plumbing {
// 	if r.apartment == nil {
// 		return nil
// 	}

// 	result := make([]*Plumbing, 0, len(r.Plumbing))
// 	for _, pID := range r.Plumbing {
// 		p, err := r.apartment.GetPlumbingByID(pID)
// 		if err != nil {
// 			continue
// 		}
// 		result = append(result, p)
// 	}
// 	return result
// }

// // GetAppliances возвращает объекты бытовой техники комнаты, разрешая ID через индекс квартиры.
// func (r *Room) GetAppliances() []*Appliances {
// 	if r.apartment == nil {
// 		return nil
// 	}

// 	result := make([]*Appliances, 0, len(r.Appliances))
// 	for _, aID := range r.Appliances {
// 		a, err := r.apartment.GetAppliancesByID(aID)
// 		if err != nil {
// 			continue
// 		}
// 		result = append(result, a)
// 	}
// 	return result
// }

// GetWalls возвращает объекты стен комнаты, разрешая ID через индекс квартиры.
func (r *Room) GetWalls() []*Wall {
	if r.apartment == nil {
		return nil
	}

	result := make([]*Wall, 0, len(r.Walls))
	for _, wID := range r.Walls {
		w, err := r.apartment.GetWallByID(wID)
		if err != nil {
			continue
		}
		result = append(result, w)
	}
	return result
}

// GetEntryDoor возвращает входную дверь в комнату, если r.Name == "hall".
// Иначе nil
func (r *Room) GetEntryDoor(ap *Apartment) *Door {
	if r.Name != RoomHall {
		return nil
	}

	for _, dID := range r.Doors {
		door, ok := ap.doorsByID[dID]
		if !ok {
			continue
		}

		if len(door.Rooms) == 1 {
			return door
		}
	}

	return nil
} 
