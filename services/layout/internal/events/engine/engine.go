package engine

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules"
)

type Engine struct {
	rules []rules.Rule
	RoomIDToRoom map[string]*entities.Room // вспомогательное поле
}

func NewEngine(rules []rules.Rule) *Engine {
	return &Engine{rules: rules, RoomIDToRoom: make(map[string]*entities.Room)}
}

func (e *Engine) PlaceDevices(apartment *entities.Apartment) (*entities.ApartmentResult, error) {
	if apartment == nil {
		return nil, fmt.Errorf("nil apartment")
	}

	if apartment.Rooms == nil {
		return nil, fmt.Errorf("nil rooms")
	}

	for _, room := range apartment.Rooms {
		if room == nil {
			return nil, fmt.Errorf("nil room")
		}
		e.RoomIDToRoom[room.ID] = room
	}
	
	res := entities.NewApartmentResult()

	for _, rule := range e.rules {
		if !rule.HasSuitableTrack(apartment) {
			continue
		}

		res.Placements = rule.Apply(apartment)
	}
	return res, nil
}
