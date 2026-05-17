package apartment

const (
	RoomLiving   = "living"
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
