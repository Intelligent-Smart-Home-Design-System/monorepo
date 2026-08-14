package apartment

import "strings"

const (
	RoomLiving       string = "livingroom"
	RoomBedroom      string = "bedroom"
	RoomKitchen      string = "kitchen"
	RoomPassage      string = "passage"
	RoomBathroom     string = "bathroom"
	RoomCabinet      string = "cabinet"
	RoomHall         string = "hallway"
	RoomCloset       string = "closet"
	RoomPorch        string = "porch"
	RoomPantry       string = "pantry"
	RoomUtility      string = "utility"
	RoomMultiPurpose string = "multipurpose"
	RoomUnknown      string = "unknown"
)

func ParseSingleRoomType(rawName string) string {
	switch rawName {
	case RoomPassage:
		return RoomPassage
	case RoomLiving, "living":
		return RoomLiving
	case RoomBedroom:
		return RoomBedroom
	case RoomKitchen:
		return RoomKitchen
	case RoomBathroom, "bath":
		return RoomBathroom
	case RoomCabinet:
		return RoomCabinet
	case RoomHall, "hall":
		return RoomHall
	case RoomCloset:
		return RoomCloset
	case RoomPantry:
		return RoomPantry
	case RoomUtility:
		return RoomUtility
	case RoomPorch:
		return RoomPorch
	}

	switch {
	case strings.Contains(rawName, "bedroom"), strings.Contains(rawName, "bdrm"):
		return RoomBedroom

	case strings.Contains(rawName, "bath"), strings.Contains(rawName, "toil"), strings.Contains(rawName, "restroom"), strings.Contains(rawName, "wc"):
		return RoomBathroom

	case strings.Contains(rawName, "w.i.c."), strings.Contains(rawName, "closet"), strings.Contains(rawName, "clo."):
		return RoomCloset

	case strings.Contains(rawName, "kitchen"), strings.Contains(rawName, "dining"):
		return RoomKitchen

	case strings.Contains(rawName, "living"), strings.Contains(rawName, "great room"):
		return RoomLiving

	case strings.Contains(rawName, "pantry"):
		return RoomPantry

	case strings.Contains(rawName, "util"), strings.Contains(rawName, "boiler"):
		return RoomUtility

	case strings.Contains(rawName, "porch"), strings.Contains(rawName, "balcony"), strings.Contains(rawName, "terrace"):
		return RoomPorch

	case strings.Contains(rawName, "passage"):
		return RoomPassage

	case strings.Contains(rawName, "hall"), strings.Contains(rawName, "entry"), strings.Contains(rawName, "foyer"):
		return RoomHall

	case strings.Contains(rawName, "cabinet"), strings.Contains(rawName, "office"), strings.Contains(rawName, "study"):
		return RoomCabinet

	default:
		return RoomUnknown
	}
}

func ParseRoomTypes(rawName string) []string {
	if !strings.Contains(rawName, "/") {
		return []string{ParseSingleRoomType(rawName)}
	}

	parts := strings.Split(rawName, "/")

	var res []string

	for _, part := range parts {
		singleType := ParseSingleRoomType(part)
		
		if singleType != RoomUnknown {
			res = append(res, singleType)
		}
	}

	if len(res) == 0 {
		return []string{RoomUnknown}
	}

	return res
}

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
