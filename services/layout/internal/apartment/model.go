<<<<<<< HEAD
package apartment

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type Apartment struct {
	Walls      []Wall       `json:"walls"`
	Doors      []Door       `json:"door"`
	Windows    []Window     `json:"windows"`
	Rooms      []Room       `json:"rooms"`
	Furniture  []Furniture  `json:"furniture"`
	Plumbing   []Plumbing   `json:"plumbing"`
	Appliances []Appliances `json:"appliances"`

	roomsByName    map[string][]*Room
	wallsByID      map[string]*Wall
	windowByID     map[string]*Window
	doorsByID      map[string]*Door
	furnitureByID  map[string]*Furniture
	plumbingByID   map[string]*Plumbing
	appliancesByID map[string]*Appliances
}

type Wall struct {
	ID     string        `json:"id"`
	Points []point.Point `json:"points"` // начальная и конечная точки
	Width  float64       `json:"width"`
}

type Door struct {
	ID     string        `json:"id"`
	Points []point.Point `json:"points"`
	Width  float64       `json:"width"`
	Rooms  []string      `json:"rooms"` // ID комнат, которые соединяет дверь
}

type Window struct {
	ID     string        `json:"id"`
	Points []point.Point `json:"points"`
	Width  float64       `json:"width"`
	Rooms  []string      `json:"rooms"`
}

type Furniture struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Points []point.Point `json:"points"`
	Room  string      `json:"room"`
}

type Plumbing struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Points []point.Point `json:"points"`
	Room   string        `json:"room"`
}

type Appliances struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Points []point.Point `json:"points"`
	Room   string        `json:"room"`
}

type Room struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Area       []point.Point `json:"area"`
	AreaM2     float64       `json:"area_m2"`
	Windows    []string      `json:"windows"`    // ID окон
	Doors      []string      `json:"doors"`      // ID дверей
	Walls      []string      `json:"walls"`      // ID стен
	Furniture  []string      `json:"furniture"`  // ID мебели
	Plumbing   []string      `json:"plumbing"`   // ID сантехники (унитаз, раковина, ванна, душ)
	Appliances []string      `json:"appliances"` // ID бытовой техники (стиральная машина, посудомоечная машина)

	// back-reference to parent apartment for resolving IDs to objects
	apartment *Apartment
	Center    *point.Point
}
=======
package apartment

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
)

type Apartment struct {
	Walls      []Wall       `json:"walls"`
	Doors      []Door       `json:"door"`
	Windows    []Window     `json:"windows"`
	Rooms      []Room       `json:"rooms"`
	Furniture  []Furniture  `json:"furniture"`
	Plumbing   []Plumbing   `json:"plumbing"`
	Appliances []Appliances `json:"appliances"`

	roomsByName    map[string][]*Room
	wallsByID      map[string]*Wall
	windowByID     map[string]*Window
	doorsByID      map[string]*Door
	furnitureByID  map[string]*Furniture
	plumbingByID   map[string]*Plumbing
	appliancesByID map[string]*Appliances
}

type Wall struct {
	ID     string        `json:"id"`
	Points []point.Point `json:"points"` // начальная и конечная точки
	Width  float64       `json:"width"`
}

type Door struct {
	ID     string        `json:"id"`
	Points []point.Point `json:"points"`
	Width  float64       `json:"width"`
	Rooms  []string      `json:"rooms"` // ID комнат, которые соединяет дверь
}

type Window struct {
	ID     string        `json:"id"`
	Points []point.Point `json:"points"`
	Width float64       `json:"width"`
	Rooms  []string      `json:"rooms"`
}

type Furniture struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Points []point.Point `json:"points"`
	Rooms  []string      `json:"rooms"`
}

type Plumbing struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Points []point.Point `json:"points"`
	Room   string        `json:"room"`
}

type Appliances struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Points []point.Point `json:"points"`
	Room   string        `json:"room"`
}

type Room struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Area       []point.Point `json:"area"`
	AreaM2     float64       `json:"area_m2"`
	Windows    []string      `json:"windows"`    // ID окон
	Doors      []string      `json:"doors"`      // ID дверей
	Walls      []string      `json:"walls"`      // ID стен
	Furniture  []string      `json:"furniture"`  // ID мебели
	Plumbing   []string      `json:"plumbing"`   // ID сантехники (унитаз, раковина, ванна, душ)
	Appliances []string      `json:"appliances"` // ID бытовой техники (стиральная машина, посудомоечная машина)

	// back-reference to parent apartment for resolving IDs to objects
	apartment *Apartment
	Center    *point.Point
}
>>>>>>> 4bf54f8 (hz)
