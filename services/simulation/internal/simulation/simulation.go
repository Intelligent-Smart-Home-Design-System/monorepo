package simulation

import (
	"context"
	"log/slog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/decoder"
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
	fieldData, err := s.fetcher.GetField()
	if err != nil {
		return err
	}

	field, err := decoder.ParseField(fieldData)
	if err != nil {
		return err
	}

	entitiesData, err := s.fetcher.GetEntities()
	if err != nil {
		return err
	}

	entities, err := decoder.ParseEntities(entitiesData)
	if err != nil {
		return err
	}

	simEngine := engine.NewSimEngine(field)
	engineEventsQueue := simEngine.GetQueue()

	simEngine.InitEntities(entities)
	simEngine.InitProcesses()

	go func() {
		err = simEngine.Run()
		slog.Error("cannot initialize engine for simulation")
	}()

	for {
		eventsData, err := s.fetcher.GetEvents()
		if err != nil {
			return err
		}

		events, err := decoder.ParseEvents(eventsData)
		if err != nil {
			return err
		}

		for _, event := range events {
			engineEventsQueue <- event
		}
	}
}
