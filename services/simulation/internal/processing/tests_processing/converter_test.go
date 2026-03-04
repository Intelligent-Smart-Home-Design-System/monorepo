package tests_converter

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/converter"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/devices"
)

type stubEnginePort struct {
	outChan chan api.EventOutDTO
}

func (s *stubEnginePort) GetOutChan() chan api.EventOutDTO {
	return s.outChan
}

func (s *stubEnginePort) UpdateField(x, y int, cell field.Cell) error {
	return nil
}

//проверка парсинга лампы
func TestEntitiesFromDTO_Lamp(t *testing.T) {
	engineStub := &stubEnginePort{}

	lampJSON := []byte(`{"id":"lamp_1","turned_on":false,"delay":1.0,"receivers":[]}`)

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "lamp_1",
			Info: lampJSON,
		},
	}

	entitiesMap, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lampEntity, ok := entitiesMap["lamp_1"].(*devices.Lamp)
	if !ok {
		t.Fatalf("expected *Lamp type, got %T", entitiesMap["lamp_1"])
	}

	if lampEntity.TurnedOn != false {
		t.Errorf("expected TurnedOn false, got %v", lampEntity.TurnedOn)
	}

	if lampEntity.Delay != 1.0 {
		t.Errorf("expected Delay 1.0, got %v", lampEntity.Delay)
	}

	if len(lampEntity.Receivers) != 0 {
		t.Errorf("expected none receivers, got %v", lampEntity.Receivers)
	}
}

//проверка парсинга переключателя
func TestEntitiesFromDTO_LampSwitcher(t *testing.T) {
	engineStub := &stubEnginePort{}

	switcherJSON := []byte(`{"id":"lampSwitcher_1","turned_on":true,"delay":0.5,"receivers":["lamp_1"]}`)

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "lampSwitcher_1",
			Info: switcherJSON,
		},
	}

	entitiesMap, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	switcherEntity, ok := entitiesMap["lampSwitcher_1"].(*devices.LampSwitcher)
	if !ok {
		t.Fatalf("expected *LampSwitcher type, got %T", entitiesMap["lampSwitcher_1"])
	}

	if switcherEntity.TurnedOn != true {
		t.Errorf("expected TurnedOn true, got %v", switcherEntity.TurnedOn)
	}

	if switcherEntity.Delay != 0.5 {
		t.Errorf("expected Delay 0.5, got %v", switcherEntity.Delay)
	}

	if len(switcherEntity.Receivers) != 1 || switcherEntity.Receivers[0] != "lamp_1" {
		t.Errorf("expected receivers ['lamp_1'], got %v", switcherEntity.Receivers)
	}
}

//проверка парсинга неизвестного устройства
func TestEntitiesFromDTO_InvalidType(t *testing.T) {
	engineStub := &stubEnginePort{}

	entitiesDTO := []api.EntityDTO{
		{
			ID:   "unknown_1",
			Info: []byte(`{}`),
		},
	}

	_, err := converter.EntitiesFromDTO(entitiesDTO, engineStub)
	if err == nil {
		t.Fatal("expected error for invalid entity type, got nil")
	}
	if err != converter.ErrorInvalidFormat {
		t.Fatalf("expected ErrorInvalidFormat, got %v", err)
	}
}

//проверка парсинга поля
func TestFieldFromDTO(t *testing.T) {
	fieldDTO := api.FieldDTO{
		Width:  2,
		Height: 2,
		Cells: [][]*api.CellDTO{
			{
				{X: 0, Y: 0, Condition: false},
				{X: 0, Y: 1, Condition: true},
			},
			{
				{X: 1, Y: 0, Condition: true},
				{X: 1, Y: 1, Condition: false},
			},
		},
	}

	simField := converter.FieldFromDTO(fieldDTO)
	if simField.Width != 2 || simField.Height != 2 {
		t.Errorf("expected Width=2, Height=2, got Width=%d, Height=%d", simField.Width, simField.Height)
	}

	expected := [][]bool{
		{false, true},
		{true, false},
	}

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if simField.Cells[i][j].Condition != expected[i][j] {
				t.Errorf("cell[%d][%d] expected Condition=%v, got %v", i, j, expected[i][j], simField.Cells[i][j].Condition)
			}
		}
	}
}