package rules

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

type Rule interface {
	Type() string

	Apply(zonedAp *apartment.ZonedApartment, levelNum string, 
		deviceRooms []string, maxCount int, layout *apartment.Layout) error
}
