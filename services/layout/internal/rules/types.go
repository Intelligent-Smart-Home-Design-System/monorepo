package rules

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/device"
)

type Rule interface {
	// GetType возвращает тип устройства, относящегося к этому правилу
	GetType() string

	// Apply возвращает мапу, которая по roomID и deviceID выдает расставленное устройство
	// (объект структуры Placement). Через Apply устройство расставляется во всех нужных
	// местах в каждой комнате.
	Apply(ap *apartment.Apartment) map[string]map[string]*device.Placement
}
