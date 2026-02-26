package rules

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
)

type Rule interface {

	HasSuitableTrack(apartment *entities.Apartment) bool	// скорее всего, не понадобится
															// после введения конфига треков

	Apply(apartment *entities.Apartment) map[string]map[string]*entities.Placement
}
