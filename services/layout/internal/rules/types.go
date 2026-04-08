package rules

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
)

type Rule interface {
	// GetType возвращает тип устройства, относящегося к этому правилу
	GetType() string

	// Apply расставляет устройство в квартире
	Apply(apartment *entities.Apartment, deviceRooms []string, apartmentLayout *entities.ApartmentLayout) error
}
