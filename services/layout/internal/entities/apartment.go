package entities

type Apartment struct {
	Walls   []Wall   `json:"walls"`
	Doors   []Door   `json:"door"`
	Windows []Window `json:"windows"`
	Rooms   []Room   `json:"rooms"`
}

type Point struct {
	X float64
	Y float64
}

func NewVector(p1, p2 Point) Point {
	return Point{p2.X - p1.X, p2.Y - p1.Y}
}

type Segment struct {
	LeftPoint  Point
	RightPoint Point
}

type Wall struct {
	ID     string  `json:"id"`
	Points []Point `json:"points"` // начальная и конечная точки
	Width  float64 `json:"width"`
}

type Door struct {
	ID     string   `json:"id"`
	Points []Point  `json:"points"`
	Width  float64  `json:"width"`
	Rooms  []string `json:"rooms"` // ID комнат, которые соединяет дверь
}

type Window struct {
	ID     string   `json:"id"`
	Points []Point  `json:"points"`
	Height float64  `json:"height"`
	Rooms  []string `json:"rooms"`
}

type Room struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Area    []Point  `json:"area"`
	AreaM2  float64  `json:"area_m2"`
	Windows []string `json:"windows"` // ID окон
	Doors   []string `json:"doors"`   // ID дверей
	Walls   []string `json:"walls"`   // ID стен
}

type ApartmentLayout struct {
	Placements map[string]map[string]*Placement // roomID -> deviceType -> devicePlacement
	// То есть по roomID получаем мапу между
	// типом устройства и его расстановкой

	// в дальнейшем необходимо будет хранить доп поля в этой структуре (для других модулей)
}

func NewApartmentResult() *ApartmentLayout {
	return &ApartmentLayout{Placements: make(map[string]map[string]*Placement)}
}
