package simulations

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
)

//=====Stubs=====
type StubFetcher struct {
	simIDs       []string
	entities     map[string][]api.EntityDTO
	dependencies map[string]map[string][]api.ActionDTO
	fields       map[string]api.FieldDTO
	events       map[string][]api.EventInDTO
	mu           sync.Mutex
}

func (s *StubFetcher) GetSimulationsID() []string {
	return s.simIDs
}

func (s *StubFetcher) GetEntities() (map[string][]api.EntityDTO, error) {
	return s.entities, nil
}

func (s *StubFetcher) GetDependencies() (map[string]map[string][]api.ActionDTO, error) {
	return s.dependencies, nil
}

func (s *StubFetcher) GetFields() (map[string]api.FieldDTO, error) {
	return s.fields, nil
}

func (s *StubFetcher) GetEvents() (map[string][]api.EventInDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	evCopy := make(map[string][]api.EventInDTO, len(s.events))
	for simID, events := range s.events {
		evCopy[simID] = append([]api.EventInDTO(nil), events...)
		s.events[simID] = nil
	}

	return evCopy, nil
}

type StubSender struct {
	events []api.EventOutDTO
	mu     sync.Mutex
}

func (s *StubSender) Run() {
}

func (s *StubSender) AddEvent(dto api.EventOutDTO) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, dto)
}

func (s *StubSender) Send(dto api.EventOutDTO) {
}

//=====Helper=====
func (s *StubSender) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.events)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func event(id string, state bool) api.EventInDTO {
	data, _ := json.Marshal(map[string]bool{"turn_on": state,})

	return api.EventInDTO{
		EntityID: id,
		Info:     data,
	}
}

//=====Tests=====
//Тест проверки корректности работы программы в стандартном случае
//  - очередь событий задана и не меняется со временем
func TestSimulation_Default(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	simID := "sim1"

	switch1 := api.EntityDTO{
		ID: "lampSwitcher_1",
		Info: mustJSON(map[string]any{"id": "lampSwitcher_1", "delay": 1.0}),
	}
	switch2 := api.EntityDTO{
		ID: "lampSwitcher_2",
		Info: mustJSON(map[string]any{"id": "lampSwitcher_2", "delay": 0.3}),
	}
	lamp1 := api.EntityDTO{
		ID: "lamp_1",
		Info: mustJSON(map[string]any{"id": "lamp_1", "delay": 0.5}),
	}
	lamp2 := api.EntityDTO{
		ID: "lamp_2",
		Info: mustJSON(map[string]any{"id": "lamp_2", "delay": 1.0}),
	}

	deps := map[string]map[string][]api.ActionDTO{
		simID: {
			"lampSwitcher_1": {	{ID: "lamp_1"} },
			"lampSwitcher_2": {	{ID: "lamp_1"},	{ID: "lamp_2"} },
		},
	}

	events := map[string][]api.EventInDTO{
		simID: {
			{EntityID: "step_init"},
			event("lampSwitcher_1", true),
			{EntityID: "step_tick"},
			event("lampSwitcher_2", true),
			{EntityID: "step_tick"},
			event("lampSwitcher_1", false),
			{EntityID: "step_tick"},
			{EntityID: "step_tick"},
		},
	}

	fetcher := &StubFetcher{
		simIDs:       []string{simID},
		entities:     map[string][]api.EntityDTO{simID: {switch1, switch2, lamp1, lamp2}},
		dependencies: deps,
		fields:       map[string]api.FieldDTO{simID: {}},
		events:       events,
	}

	sender := &StubSender{}

	sim := NewSimulation(fetcher, sender)

	err := sim.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := sim.Run(ctx); err != nil && err != context.Canceled {
			t.Logf("Simulations.Run error: %v", err)
		}
	}()
	
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sender.Count() >= 7 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if sender.Count() != 7 {
		t.Fatal("expected 7 events, got not 7")
	}
}

//Тест проверки корректности работы программы в стандартном случае
//  - очередь событий меняется со временем
func TestSimulation_UserIntervention(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	simID := "sim2"

	switch1 := api.EntityDTO{
		ID:   "lampSwitcher_1",
		Info: mustJSON(map[string]any{"id": "lampSwitcher_1", "delay": 0.0}),
	}
	switch2 := api.EntityDTO{
		ID:   "lampSwitcher_2",
		Info: mustJSON(map[string]any{"id": "lampSwitcher_2", "delay": 0.3}),
	}
	lamp1 := api.EntityDTO{
		ID:   "lamp_1",
		Info: mustJSON(map[string]any{"id": "lamp_1", "delay": 0.5}),
	}
	lamp2 := api.EntityDTO{
		ID:   "lamp_2",
		Info: mustJSON(map[string]any{"id": "lamp_2", "delay": 0.5}),
	}

	deps := map[string]map[string][]api.ActionDTO{
		simID: {
			"lampSwitcher_1": {{ID: "lamp_1"}},
			"lampSwitcher_2": {{ID: "lamp_2"}},
		},
	}

	events := map[string][]api.EventInDTO{
		simID: {
			{EntityID: "step_init"},
		},
	}

	fetcher := &StubFetcher{
		simIDs:       []string{simID},
		entities:     map[string][]api.EntityDTO{simID: {switch1, switch2, lamp1, lamp2}},
		dependencies: deps,
		fields:       map[string]api.FieldDTO{simID: {}},
		events:       events,
	}

	sender := &StubSender{}

	sim := NewSimulation(fetcher, sender)

	err := sim.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := sim.Run(ctx); err != nil && err != context.Canceled {
			t.Logf("Simulations.Run error: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	fetcher.mu.Lock()
	fetcher.events[simID] = append(fetcher.events[simID], api.EventInDTO{
		EntityID: "lampSwitcher_1",
		Info:     mustJSON(map[string]bool{"turn_on": true}),
	})
	fetcher.events[simID] = append(fetcher.events[simID], api.EventInDTO{EntityID: "step_tick"})
	fetcher.mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	fetcher.mu.Lock()
	fetcher.events[simID] = append(fetcher.events[simID], api.EventInDTO{
		EntityID: "lampSwitcher_1",
		Info:     mustJSON(map[string]bool{"turn_on": false}),
	})
	fetcher.events[simID] = append(fetcher.events[simID], api.EventInDTO{EntityID: "step_tick"})
	fetcher.mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	fetcher.mu.Lock()
	fetcher.events[simID] = append(fetcher.events[simID], api.EventInDTO{
		EntityID: "lampSwitcher_2",
		Info:     mustJSON(map[string]bool{"turn_on": true}),
	})
	fetcher.events[simID] = append(fetcher.events[simID], api.EventInDTO{EntityID: "step_tick"})
	fetcher.mu.Unlock()

	time.Sleep(500 * time.Millisecond)

	var lamp1State, lamp2State bool
	for _, e := range sender.events {
		if e.EntityID == "lamp_1" {
			var out map[string]bool
			_ = json.Unmarshal(e.Info, &out)
			lamp1State = out["turn_on"]
		}
		if e.EntityID == "lamp_2" {
			var out map[string]bool
			_ = json.Unmarshal(e.Info, &out)
			lamp2State = out["turn_on"]
		}
	}

	if lamp1State != false {
		t.Fatalf("lamp_1 expected OFF, got ON")
	}
	if lamp2State != true {
		t.Fatalf("lamp_2 expected ON, got OFF")
	}
}
