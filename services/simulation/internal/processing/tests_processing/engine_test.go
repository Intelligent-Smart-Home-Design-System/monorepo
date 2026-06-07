package tests_processing

import (
	"errors"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/fschuetz04/simgo"
)

// =====Stubs=====
type stubEntity struct {
	id        string
	receivers []string
	setCalled bool
}

func (s *stubEntity) GetID() string {
	return s.id
}

func (s *stubEntity) GetReceiversID() []string {
	return s.receivers
}

func (s *stubEntity) SetReceivers(actions []api.EdgeDTO) {
	s.setCalled = true

	var ids []string
	for _, a := range actions {
		ids = append(ids, a.ToID)
	}

	s.receivers = ids
}

type stubEntityWithProcess struct {
	stubEntity
	handleErr   error
	handleCalls int
}

func (s *stubEntityWithProcess) GetProcessFunc() func(process simgo.Process) {
	return func(process simgo.Process) {}
}

func (s *stubEntityWithProcess) HandleInDTO(dto []byte) error {
	s.handleCalls++
	return s.handleErr
}

func (s *stubEntityWithProcess) HandleOutDTO(dto []byte) {
}

func (s *stubEntityWithProcess) Process(process simgo.Process) {
}

func (s *stubEntityWithProcess) GetOutCh() chan []byte {
	return make(chan []byte)
}

// =====Tests=====
// Тест проверки функции CheckCircleDependencies()
func TestCheckCircleDependencies(t *testing.T) {
	tests := []struct {
		name     string
		entities map[string]entities.Entity
		want     bool
	}{
		{
			name: "no cycle",
			entities: map[string]entities.Entity{
				"a": &stubEntity{id: "a", receivers: []string{"b"}},
				"b": &stubEntity{id: "b", receivers: []string{}},
			},
			want: false,
		},
		{
			name: "simple cycle",
			entities: map[string]entities.Entity{
				"a": &stubEntity{id: "a", receivers: []string{"b"}},
				"b": &stubEntity{id: "b", receivers: []string{"a"}},
			},
			want: true,
		},
		{
			name: "self cycle",
			entities: map[string]entities.Entity{
				"a": &stubEntity{id: "a", receivers: []string{"a"}},
			},
			want: true,
		},
		{
			name:     "empty entities",
			entities: map[string]entities.Entity{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := engine.NewSimEngine(1.0)
			e.IDToEntity = tt.entities

			got := e.CheckCircleDependencies()
			if got != tt.want {
				t.Errorf("CheckCircleDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Тест проверки функции HandleEvent()
func TestHandleEvent(t *testing.T) {
	tests := []struct {
		name        string
		entity      *stubEntityWithProcess
		expectCalls int
		expectEnq   bool
	}{
		{
			name: "handle success",
			entity: &stubEntityWithProcess{
				stubEntity: stubEntity{id: "a", receivers: []string{"b"}},
			},
			expectCalls: 1,
			expectEnq:   true,
		},
		{
			name: "handle returns error",
			entity: &stubEntityWithProcess{
				stubEntity: stubEntity{id: "a", receivers: []string{}},
				handleErr:  errors.New("fail"),
			},
			expectCalls: 1,
			expectEnq:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := engine.NewSimEngine(1.0)
			e.IDToEntity["a"] = tt.entity
			e.IDToEntity["b"] = &stubEntity{id: "b"}
			event := api.EventDTO{EntityID: "a"}

			e.HandleEvent(event)

			if tt.entity.handleCalls != tt.expectCalls {
				t.Errorf("handle calls = %v, want %v", tt.entity.handleCalls, tt.expectCalls)
			}
		})
	}
}

// Тест корректного поведения функции Run() при закрытии канала
func TestRun_ChannelClosed(t *testing.T) {
	t.Run("channel closed", func(t *testing.T) {
		e := engine.NewSimEngine(1.0)
		close(e.GetInChan())

		e.Step()
		t.Logf("Closed channel correct working")
	})
}
