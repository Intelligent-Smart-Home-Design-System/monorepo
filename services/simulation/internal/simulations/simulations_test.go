package simulations

import (
	"errors"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/fetcher"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/sender"
)

// =====Stubs=====
type stubFetcher struct {
	simIDs       []string
	fields       map[string]api.FieldDTO
	entities     map[string][]api.EntityDTO
	dependencies map[string]map[string][]api.EdgeDTO
	events       map[string][]engine.EventIn
	err          error
}

func (s *stubFetcher) GetSimulationsID() []string {
	return s.simIDs
}

func (s *stubFetcher) GetFields() (map[string]api.FieldDTO, error) {
	return s.fields, s.err
}

func (s *stubFetcher) GetEntities() (map[string][]api.EntityDTO, error) {
	return s.entities, s.err
}

func (s *stubFetcher) GetDependencies() (map[string]map[string][]api.EdgeDTO, error) {
	return s.dependencies, s.err
}

func (s *stubFetcher) GetEvents() (map[string][]engine.EventIn, error) {
	return s.events, s.err
}

type stubSender struct {
	events []api.EventOutDTO
}

func (s *stubSender) Run() {
}

func (s *stubSender) AddEvent(e api.EventOutDTO) {
	s.events = append(s.events, e)
}

func (s *stubSender) Send(e api.EventOutDTO) {
	s.events = append(s.events, e)
}

// =====Helper=====
func newTestSimulations(fetcher fetcher.Fetcher, sender sender.Sender) *Simulations {
	return NewSimulation(fetcher, sender)
}

// =====Tests=====
// Тест проверки инициализации
func TestNewSimulation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"create simulation struct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &stubFetcher{}
			sender := &stubSender{}

			s := NewSimulation(fetcher, sender)

			if s == nil {
				t.Fatalf("simulation is nil")
			}
			if s.fetcher != fetcher {
				t.Errorf("fetcher not set")
			}
			if s.sender != sender {
				t.Errorf("sender not set")
			}
		})
	}
}

// Тест проверки функции InitEngines()
func TestInitEngines(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"init engines successfully"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &stubFetcher{
				simIDs: []string{"sim1"},
				fields: map[string]api.FieldDTO{
					"sim1": {},
				},
				entities: map[string][]api.EntityDTO{
					"sim1": {
						{
							ID:   "lamp_1",
							Info: []byte("{}"),
						},
					},
				},
				dependencies: map[string]map[string][]api.EdgeDTO{
					"sim1": {
						"lamp_1": {},
					},
				},
			}
			sender := &stubSender{}
			s := newTestSimulations(fetcher, sender)

			err := s.InitEngines()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(s.IDToEngine) != 1 {
				t.Errorf("engine not created")
			}
		})
	}
}

// Тест проверки функции InitEngines()
func TestInitEngines_FetchFieldsError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"fields error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &stubFetcher{
				simIDs: []string{"sim1"},
				err:    errors.New("fetch error"),
			}
			sender := &stubSender{}
			s := newTestSimulations(fetcher, sender)

			err := s.InitEngines()

			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

// Тест проверки функции GetEnginesInChan()
func TestGetEnginesInChan(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"set in channels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &stubFetcher{}
			sender := &stubSender{}
			s := newTestSimulations(fetcher, sender)
			engine := engine.NewSimEngine()
			s.IDToEngine["sim1"] = engine

			s.GetEnginesInChan()

			if s.IDToEventInChan["sim1"] == nil {
				t.Errorf("channel not set")
			}
		})
	}
}

// Тест проверки функции GetEnginesOutChan()
func TestGetEnginesOutChan(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"set out channels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &stubFetcher{}
			sender := &stubSender{}
			s := newTestSimulations(fetcher, sender)
			engine := engine.NewSimEngine()
			s.IDToEngine["sim1"] = engine

			s.GetEnginesOutChan()

			if s.IDToEventOutChan["sim1"] == nil {
				t.Errorf("channel not set")
			}
		})
	}
}

// Тест проверки функции Stop()
func TestStop(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"close event channels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &stubFetcher{}
			sender := &stubSender{}
			s := newTestSimulations(fetcher, sender)
			ch := make(chan engine.EventIn)
			s.IDToEventInChan["sim1"] = ch

			s.Stop()

			_, ok := <-ch
			if ok {
				t.Errorf("channel not closed")
			}
		})
	}
}
