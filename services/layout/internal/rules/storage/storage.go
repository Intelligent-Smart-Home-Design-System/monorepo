package storage

import (
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/lighting"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/security"
)

// Storage является хранилищем правил для расстановки устройств
type Storage struct {
	rules map[string]rules.Rule
}

func NewStorage() *Storage {
	return &Storage{rules: make(map[string]rules.Rule)}
}

func (s *Storage) LoadRule(rule rules.Rule) {
	s.rules[rule.GetType()] = rule
}

func (s *Storage) GetRule(deviceType string) (rules.Rule, error) {
	rule, ok := s.rules[deviceType]
	if !ok {
		return nil, fmt.Errorf("failed to get rule for device %s", deviceType)
	}

	return rule, nil
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
