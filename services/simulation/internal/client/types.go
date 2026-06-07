package client

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"

// SimulationService представляет интерфейс для управления симуляцией
type SimulationService interface {
	// Start запускает симуляцию с заданным идентификатором запроса и payload, возвращая ошибку в случае неудачи.
	Start(reqID string, payload api.SimulationStartPayload) error

	// Tick выполняет один шаг симуляции с заданным идентификатором запроса и payload, возвращая результат шага или ошибку в случае неудачи.
	Tick(reqID string, payload api.SimulationTickPayload) (*api.SimulationStepPayload, error)

	// Stop останавливает симуляцию с заданным идентификатором запроса, возвращая ошибку в случае неудачи.
	Stop(reqID string) error
}
