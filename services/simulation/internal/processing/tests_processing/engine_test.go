package tests_processing

import (
	"context"
	"errors"
	"testing"

	"github.com/fschuetz04/simgo"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
)

//=====Stubs=====
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

func (s *stubEntity) SetReceivers(actions []api.ActionDTO) {
	s.setCalled = true
	var ids []string
	for _, a := range actions {
		ids = append(ids, a.ID)
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

func (s *stubEntityWithProcess) HandleOutDTO(out any) error {
	return nil
}

func (s *stubEntityWithProcess) Process(process simgo.Process) {
}

func (s *stubEntityWithProcess) GetOutCh() chan []byte {
	return make(chan []byte)
}

//=====Helper=====
func newTestField(height, width int) *field.Field {
	cells := make([][]*field.Cell, height+1)
	for i := range cells {
		cells[i] = make([]*field.Cell, width+1)
		for j := range cells[i] {
			cells[i][j] = &field.Cell{}
		}
	}
	return &field.Field{
		Height: height,
		Width:  width,
		Cells:  cells,
	}
}

//=====Tests=====
//Тест проверки функции CheckCircleDependencies()
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
			e := engine.NewSimEngine()
			e.IDToEntity = tt.entities
			got := e.CheckCircleDependencies()
			if got != tt.want {
				t.Errorf("CheckCircleDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}

//Тест проверки функции HandleEvent()
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
			e := engine.NewSimEngine()
			e.IDToEntity["a"] = tt.entity
			e.IDToEntity["b"] = &stubEntity{id: "b"}
			event := api.EventInDTO{EntityID: "a"}

			e.HandleEvent(event)

			if tt.entity.handleCalls != tt.expectCalls {
				t.Errorf("handle calls = %v, want %v", tt.entity.handleCalls, tt.expectCalls)
			}

			if tt.expectEnq {
				select {
				case ev := <-e.GetInChan():
					if ev.EntityID != "b" {
						t.Errorf("expected 'b' enqueued, got %v", ev.EntityID)
					}
				default:
					t.Errorf("expected event in channel")
				}
			}
		})
	}
}

//Тест проверки функции UpdateField()
func TestUpdateField(t *testing.T) {
	tests := []struct {
		name      string
		x, y      int
		expectErr error
	}{
		{"valid update", 0, 0, nil},
		{"invalid x", -1, 0, engine.ErrorFieldInvalidParameterX},
		{"invalid y", 0, -1, engine.ErrorFieldInvalidParameterY},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := engine.NewSimEngine()
			e.SetField(newTestField(1, 1))
			cell := field.Cell{Condition: true}

			err := e.UpdateField(tt.x, tt.y, cell)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("error = %v, want %v", err, tt.expectErr)
			}

			if tt.expectErr == nil && !e.Field.Cells[tt.x][tt.y].Condition {
				t.Errorf("cell not updated")
			}
		})
	}
}

//Тест корректного поведения функции Run() при отмене контекста
func TestRun_ContextCancel(t *testing.T) {
	t.Run("context canceled", func(t *testing.T) {
		e := engine.NewSimEngine()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := e.Run(ctx)
		if err == nil {
			t.Errorf("expected context error, got nil")
		}
	})
}

//Тест корректного поведения функции Run() при закрытии канала
func TestRun_ChannelClosed(t *testing.T) {
	t.Run("channel closed", func(t *testing.T) {
		e := engine.NewSimEngine()
		close(e.GetInChan())

		ctx := context.Background()
		err := e.Run(ctx)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
