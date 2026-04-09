package simulations

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
)

// =====Stubs=====
type StubFetcher struct {
	simIDs       []string
	entities     map[string][]api.EntityDTO
	dependencies map[string]map[string][]api.ActionDTO
	fields       map[string]api.FieldDTO
	events       map[string]chan api.EventInDTO
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
	evCopy := make(map[string][]api.EventInDTO, len(s.events))
	for simID, ch := range s.events {
		evCopy[simID] = nil
		select {
		case ev := <-ch:
			evCopy[simID] = append(evCopy[simID], ev)
		}
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

// =====Helper=====
func (s *StubSender) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.events)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func stepTick(ch chan api.EventInDTO) {
    done := make(chan struct{})
    ch <- api.EventInDTO{EntityID: "step_tick", Done: done}
    <-done // блокируемся до завершения шага
}

func stepInit(ch chan api.EventInDTO) {
    done := make(chan struct{})
    ch <- api.EventInDTO{EntityID: "step_init", Done: done}
    <-done
}

func sendEventSync(ch chan api.EventInDTO, id string, state bool) {
    done := make(chan struct{})
    data, _ := json.Marshal(map[string]bool{"turn_on": state})
    
    ch <- api.EventInDTO{
        EntityID: id,
        Info:     data,
        Done:     done,
    }
    <-done
}

func (s *StubSender) Snapshot() []api.EventOutDTO {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]api.EventOutDTO, len(s.events))
	copy(out, s.events)
	return out
}

func waitForSenderEvents(t *testing.T, sender *StubSender, want int, timeout time.Duration) []api.EventOutDTO {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if sender.Count() >= want {
			return sender.Snapshot()
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timeout: expected at least %d events, got %d", want, sender.Count())
	return nil
}

func sensorStatesFrom(events []api.EventOutDTO, sensorID string) []bool {
	states := make([]bool, 0)

	for _, e := range events {
		if e.EntityID != sensorID {
			continue
		}

		var out struct {TurnOn bool `json:"turn_on"`}
		_ = json.Unmarshal(e.Info, &out)
		states = append(states, out.TurnOn)
	}

	return states
}

// =====Tests=====
// Тест проверки корректности работы программы в стандартном случае
//   - очередь событий задана и не меняется со временем
func TestSimulation_Default(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	simID := "sim1"

	switch1 := api.EntityDTO{
		ID:   "lampSwitcher_1",
		Info: mustJSON(map[string]any{"id": "lampSwitcher_1", "delay": 1.0}),
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
		Info: mustJSON(map[string]any{"id": "lamp_2", "delay": 1.0}),
	}

	deps := map[string]map[string][]api.ActionDTO{
		simID: {
			"lampSwitcher_1": {{ID: "lamp_1"}},
			"lampSwitcher_2": {{ID: "lamp_1"}, {ID: "lamp_2"}},
		},
	}

	eventsChan := make(chan api.EventInDTO, 10)

	fetcher := &StubFetcher{
		simIDs:       []string{simID},
		entities:     map[string][]api.EntityDTO{simID: {switch1, switch2, lamp1, lamp2}},
		dependencies: deps,
		fields:       map[string]api.FieldDTO{simID: {}},
		events:       map[string]chan api.EventInDTO{simID: eventsChan},
	}

	sender := &StubSender{}

	sim := NewSimulation(fetcher, sender)

	err := sim.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := sim.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("Simulations.Run error: %v", err)
		}
	}()
	
	stepInit(eventsChan)
	sendEventSync(eventsChan, "lampSwitcher_1", true)
	stepTick(eventsChan)
	sendEventSync(eventsChan, "lampSwitcher_2", true)
	stepTick(eventsChan)
	sendEventSync(eventsChan, "lampSwitcher_1", false)
	stepTick(eventsChan)
	stepTick(eventsChan)

	waitForSenderEvents(t, sender, 7, 5*time.Second)
	if sender.Count() != 7 {
		t.Fatalf("expected 7 events, got %d", sender.Count())
	}
}

// Тест проверки корректности работы программы в стандартном случае
//   - очередь событий меняется со временем
func TestSimulation_UserIntervention(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	simID := "sim2"

	switch1 := api.EntityDTO{
		ID:   "lampSwitcher_1",
		Info: mustJSON(map[string]any{"id": "lampSwitcher_1", "delay": 0.0}),
	}
	switch2 := api.EntityDTO{
		ID:   "lampSwitcher_2",
		Info: mustJSON(map[string]any{"id": "lampSwitcher_2", "delay": 0.0}),
	}
	lamp1 := api.EntityDTO{
		ID:   "lamp_1",
		Info: mustJSON(map[string]any{"id": "lamp_1", "delay": 0.0}),
	}
	lamp2 := api.EntityDTO{
		ID:   "lamp_2",
		Info: mustJSON(map[string]any{"id": "lamp_2", "delay": 0.0}),
	}

	deps := map[string]map[string][]api.ActionDTO{
		simID: {
			"lampSwitcher_1": {{ID: "lamp_1"}},
			"lampSwitcher_2": {{ID: "lamp_2"}},
		},
	}

	eventsChan := make(chan api.EventInDTO, 10)

	fetcher := &StubFetcher{
		simIDs:       []string{simID},
		entities:     map[string][]api.EntityDTO{simID: {switch1, switch2, lamp1, lamp2}},
		dependencies: deps,
		fields:       map[string]api.FieldDTO{simID: {}},
		events:       map[string]chan api.EventInDTO{simID: eventsChan},
	}

	sender := &StubSender{}

	sim := NewSimulation(fetcher, sender)

	err := sim.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := sim.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("Simulations.Run error: %v", err)
		}
	}()

	stepInit(eventsChan)
	sendEventSync(eventsChan, "lampSwitcher_1", true)
	sendEventSync(eventsChan, "lampSwitcher_1", false)
	stepTick(eventsChan)
	sendEventSync(eventsChan, "lampSwitcher_2", true)

	events := waitForSenderEvents(t, sender, 6, 2*time.Second)

	var lamp1State, lamp2State bool
	for _, e := range events {
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

// =====Tests for LightSwitchOffSensor=====
// Обычный сценарий: сенсор получил сигнал, потом завершил таймаут и выключился.
func TestSimulation_LightSwitchOffSensor_NoInterruption(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	simID := "sim_light_switch_off_1"

	sensor := api.EntityDTO{
		ID:   "lightSwitchOffSensor_1",
		Info: mustJSON(map[string]any{"id": "lightSwitchOffSensor_1", "delay": 0.5,	"timeout": 1.0,	"turned_on": false,	"receivers": []string{}}),
	}
	lamp := api.EntityDTO{
		ID: "lamp_1",
		Info: mustJSON(map[string]any{"id": "lamp_1", "delay": 0.5,	"turned_on": false}),
	}

	deps := map[string]map[string][]api.ActionDTO{
		simID: {
			"lightSwitchOffSensor_1": {{ID: "lamp_1"}},
		},
	}

	eventsChan := make(chan api.EventInDTO, 64)

	fetcher := &StubFetcher{
		simIDs:       []string{simID},
		entities:     map[string][]api.EntityDTO{simID: {sensor, lamp}},
		dependencies: deps,
		fields:       map[string]api.FieldDTO{simID: {}},
		events:       map[string]chan api.EventInDTO{simID: eventsChan},
	}

	sender := &StubSender{}

	sim := NewSimulation(fetcher, sender)
	
	err := sim.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := sim.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("Simulations.Run error: %v", err)
		}
	}()
	
	stepInit(eventsChan)
	sendEventSync(eventsChan, "lightSwitchOffSensor_1", true)
	stepTick(eventsChan)
	sendEventSync(eventsChan, "lightSwitchOffSensor_1", false)
	stepTick(eventsChan)
	stepTick(eventsChan)

	events := waitForSenderEvents(t, sender, 4, 2*time.Second)
	got := sensorStatesFrom(events, "lightSwitchOffSensor_1")
	want := []bool{true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected sequence at index %d: got=%v want=%v", i, got, want)
		}
	}
}

// Сценарий с 2 прерываниями: каждое новое срабатывание продлевает время работы сенсора.
func TestSimulation_LightSwitchOffSensor_TwoInterruptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	simID := "sim_light_switch_off_1"

	sensor := api.EntityDTO{
		ID:   "lightSwitchOffSensor_1",
		Info: mustJSON(map[string]any{"id": "lightSwitchOffSensor_1", "delay": 0.0,	"timeout": 4.0,	"turned_on": false,	"receivers": []string{}}),
	}
	lamp := api.EntityDTO{
		ID: "lamp_1",
		Info: mustJSON(map[string]any{"id": "lamp_1", "delay": 0.0,	"turned_on": false}),
	}

	deps := map[string]map[string][]api.ActionDTO{
		simID: {
			"lightSwitchOffSensor_1": {{ID: "lamp_1"}},
		},
	}

	eventsChan := make(chan api.EventInDTO, 64)

	fetcher := &StubFetcher{
		simIDs:       []string{simID},
		entities:     map[string][]api.EntityDTO{simID: {sensor, lamp}},
		dependencies: deps,
		fields:       map[string]api.FieldDTO{simID: {}},
		events:       map[string]chan api.EventInDTO{simID: eventsChan},
	}

	sender := &StubSender{}

	sim := NewSimulation(fetcher, sender)
	
	err := sim.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := sim.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("Simulations.Run error: %v", err)
		}
	}()

	stepInit(eventsChan)
	sendEventSync(eventsChan, "lightSwitchOffSensor_1", true)
	sendEventSync(eventsChan, "lightSwitchOffSensor_1", false)

	stepTick(eventsChan)

	sendEventSync(eventsChan, "lightSwitchOffSensor_1", true)
	sendEventSync(eventsChan, "lightSwitchOffSensor_1", false)

	stepTick(eventsChan)
	stepTick(eventsChan)

	sendEventSync(eventsChan, "lightSwitchOffSensor_1", true)
	sendEventSync(eventsChan, "lightSwitchOffSensor_1", false)

	stepTick(eventsChan)
	stepTick(eventsChan)
	stepTick(eventsChan)
	stepTick(eventsChan)

	events := waitForSenderEvents(t, sender, 4, 2*time.Second)
	got := sensorStatesFrom(events, "lightSwitchOffSensor_1")
	want := []bool{true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected sequence at index %d: got=%v want=%v", i, got, want)
		}
	}
}
