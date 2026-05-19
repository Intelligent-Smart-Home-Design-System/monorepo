package rules

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

type Rule interface {
	// Type возвращает тип устройства, относящегося к этому правилу
	Type() string

	// Apply расставляет устройство в квартире
	Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, layout *apartment.Layout) error
}
