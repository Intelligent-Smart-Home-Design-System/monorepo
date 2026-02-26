package layout

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/security"
)

// Пока что абстрактная функция, конечно в будущем она будет не в этом модуле
func GetApartment() *entities.Apartment {
	return &entities.Apartment{}
}

func main() {
	apartment := GetApartment()

	device_rules := []rules.Rule{}
	device_rules = append(device_rules, security.NewWaterLeakRule("1", "security"))

	engine := engine.NewEngine(device_rules)
	_, err := engine.PlaceDevices(apartment) // вся расстановка в квартире
	if err != nil {
		_ = fmt.Errorf("place algorithm error: %w", err)
	}

	// TODO: записать результаты на плане квартиры
}