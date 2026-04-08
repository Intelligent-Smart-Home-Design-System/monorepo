package rules

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
)

type Rule interface {
	// GetType возвращает тип устройства, относящегося к этому правилу
	GetType() string

	// Apply расставляет устройство в квартире
	Apply(apartmentStruct *apartment.Apartment, deviceRooms []string, apartmentLayout *apartment.ApartmentLayout) error
}
