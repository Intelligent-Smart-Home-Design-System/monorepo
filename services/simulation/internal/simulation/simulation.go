package simulation

import "context"

// связь всей логики обработки

type Simulation struct {
	// TODO: необходимые поля
}

func NewSimulation() *Simulation {
	return &Simulation{}
}

// Run запускает симуляцию. Принимает контекст для graceful shutdown.
func (s *Simulation) Run(ctx context.Context) {
	// TODO: запуск симуляции
}
