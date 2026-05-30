package storage

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/climate"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/lighting"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/media"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/security"
)

// Storage является хранилищем правил для расстановки устройств
type Storage struct {
	Rules map[string]rules.Rule
}

func NewStorage() *Storage {
	return &Storage{Rules: make(map[string]rules.Rule)}
}

func (s *Storage) LoadRule(rule rules.Rule) {
	s.Rules[rule.Type()] = rule
}

func (s *Storage) LoadAllRules() {
	s.LoadAllSecurityRules()
	s.LoadAllLightingRules()
	s.LoadAllClimateRules()
	s.LoadAllMediaRules()

	// TODO: добавить оставшиеся треки
}

func (s *Storage) LoadAllSecurityRules() {
	storageRules := []rules.Rule{
		security.NewWaterLeakRule(),
		security.NewGasLeakRule(),
		security.NewSmartLockRule(),
		security.NewSmartDoorBellRule(),
		security.NewDoorSensorRule(),
		security.NewWindowSensorRule(),
		security.NewMotionSensorRule(),
		security.NewCameraRule(),
		security.NewSmartSirenRule(),
	}

	for _, rule := range storageRules {
		s.LoadRule(rule)
	}
}

func (s *Storage) LoadAllLightingRules() {
	storageRules := []rules.Rule{
		lighting.NewSmartBulbRule(),
		lighting.NewMotionSensorRule(),
		lighting.NewIlluminationSensorRule(),
	}

	for _, rule := range storageRules {
		s.LoadRule(rule)
	}
}

func (s *Storage) LoadAllClimateRules() {
	storageRules := []rules.Rule{
		climate.NewAirConditionerRule(),
	}

	for _, rule := range storageRules {
		s.LoadRule(rule)
	}
}

func (s *Storage) LoadAllMediaRules() {
	storageRules := []rules.Rule{
		media.NewSmartTVRule(),
	}

	for _, rule := range storageRules {
		s.LoadRule(rule)
	}
}
