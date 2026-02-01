package simulation

import (
	"context"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/converter"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/fetcher"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/sender"
)

// связь всей логики обработки

type Simulation struct {
	fetcher fetcher.Fetcher
	sender  sender.Sender
}

func NewSimulation(fetcher fetcher.Fetcher, sender sender.Sender) *Simulation {
	return &Simulation{
		fetcher: fetcher,
		sender:  sender,
	}
}

// Run запускает сервис симуляции. Принимает контекст для graceful shutdown.
func (s *Simulation) Run(ctx context.Context) error {
	slog.Debug("Creating simulation data...")

	simEngine := engine.NewSimEngine()

	err := s.InitEngine(simEngine)
	if err != nil {
		return err
	}

	engineEventsInChan := simEngine.GetInChan()
	engineEventsOutChan := simEngine.GetOutChan()

	go func() { // запуск engine
		err = simEngine.Run(ctx)
		if err != nil {
			slog.Error("Error while running engine", "error", err)
		}
	}()

	go func() {
		for out := range engineEventsOutChan {
			s.sender.Send(out)
		}
	}()

	slog.Debug("Simulation started!")

	for {
		select {
		case <-ctx.Done():
			close(engineEventsInChan)
			return ctx.Err()
		default:
		}

		events, err := s.fetcher.GetEvents() // должна быть блокирующая операция
		if err != nil {
			return err
		}

		for _, event := range events {
			select {
			case <-ctx.Done():
				close(engineEventsInChan)
				return ctx.Err()
			case engineEventsInChan <- event:
			}
		}
	}
}

func (s *Simulation) InitEngine(simEngine engine.Engine) error {
	fieldData, err := s.fetcher.GetField()
	if err != nil {
		return err
	}

	simField := converter.FieldFromDTO(fieldData)
	simEngine.SetField(simField)

	entitiesData, err := s.fetcher.GetEntities()
	if err != nil {
		return err
	}

	entities, err := converter.EntitiesFromDTO(entitiesData, simEngine)
	if err != nil {
		return err
	}

	simEngine.InitEntities(entities)
	simEngine.InitProcesses()

	return nil
}
