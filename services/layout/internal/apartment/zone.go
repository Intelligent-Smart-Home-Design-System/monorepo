package apartment

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/gofrs/uuid/v5"
)

// Zone представляет зону внутри комнаты (например, непродуваемая зона вокруг кровати).
type Zone struct {
	ID     uuid.UUID     `json:"id"`
	Points []point.Point `json:"points"`
}

func NewZone(p []point.Point) *Zone {
	return &Zone{ID: uuid.Must(uuid.NewV4()), Points: p}
}

// ZonedRoom комната, обогащённая зонами после обработки правилами.
type ZonedRoom struct {
	OrigRoom         *Room
	NoWindZones      []*Zone             `json:"no_wind_zones"`
	WetZones         []*Zone             `json:"wet_zones"`
	GasZones         []*Zone             `json:"gas_zones"`
	EntryDoorZone    *Zone               `json:"entry_doors_zones"`
	HighTrafficZones []*Zone             `json:"high_traffic_zones"`
	WindowZones      []*Zone             `json:"window_zones"`
	ViewedZones      []*Zone             `json:"viewed_zones"`
	SirenZones       []*Zone             `json:"siren_zones"`
	PollutionZones   []*Zone             `json:"pollution_zones"`
	RestrictedZones  []*Zone             `json:"restricted_zones"`
	ACAvailableWalls map[string]struct{} // nil = все стены доступны
}

func NewZonedRoom(r *Room) *ZonedRoom {
	return &ZonedRoom{OrigRoom: r}
}

// GetFurniture возвращает мебель оригинальной комнаты.
func (zr *ZonedRoom) GetFurniture() []*Furniture {
	if zr.OrigRoom == nil {
		return nil
	}
	return zr.OrigRoom.GetFurniture()
}

// GetPlumbing возвращает сантехнику оригинальной комнаты
func (zr *ZonedRoom) GetPlumbing() []*Plumbing {
	if zr.OrigRoom == nil {
		return nil
	}
	return zr.OrigRoom.GetPlumbing()
}

// GetAppliances возвращает бытовую технику оригинальной комнаты
func (zr *ZonedRoom) GetAppliances() []*Appliances {
	if zr.OrigRoom == nil {
		return nil
	}
	return zr.OrigRoom.GetAppliances()
}

// GetWalls возвращает стены оригинальной комнаты
func (zr *ZonedRoom) GetWalls() []*Wall {
	if zr.OrigRoom == nil {
		return nil
	}
	return zr.OrigRoom.GetWalls()
}

// ZonedApartment квартира, обогащённая зонами после обработки правилами.
type ZonedApartment struct {
	OrigAp     *Apartment
	ZonedRooms []*ZonedRoom
}

func NewZonedApartment(ap *Apartment) *ZonedApartment {
	return &ZonedApartment{OrigAp: ap}
}

// Build создаёт ZonedApartment из JSON-десериализованной Apartment.
// Вызывает Index() для построения внутренних индексов и создаёт
// ZonedRoom для каждой комнаты, готовую к обогащению правилами.
func Build(ap *Apartment) *ZonedApartment {
	ap.Index()

	zoned := NewZonedApartment(ap)
	zoned.ZonedRooms = make([]*ZonedRoom, 0, len(ap.Rooms))

	for i := range ap.Rooms {
		zoned.ZonedRooms = append(zoned.ZonedRooms, NewZonedRoom(&ap.Rooms[i]))
	}

	return zoned
}

// ContainsPoint проверяет, находится ли точка внутри зоны.
func (z *Zone) ContainsPoint(p point.Point) bool {
	if z == nil {
		return false
	}

	return point.IsPointInPolygon(p, z.Points)
}
