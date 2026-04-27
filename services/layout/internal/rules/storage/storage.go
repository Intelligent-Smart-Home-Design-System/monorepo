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
	storage := &Storage{
		rules: make(map[string]rules.Rule),
	}
	storage.LoadRule(security.NewWaterLeakRule())
	storage.LoadRule(lighting.NewSmartBulbRule())
	storage.LoadRule(lighting.NewMotionSensorRule())
	storage.LoadRule(lighting.NewIlluminationSensorRule())
	// TODO: загрузить все остальные правила для расстановки устройств

	return storage
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
