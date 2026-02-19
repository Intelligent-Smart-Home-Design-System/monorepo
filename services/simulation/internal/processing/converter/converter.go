package converter

import (
	"errors"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/devices"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
)

// EntitiesFromDTO парсит данные о сущностях и возвращает map[string]*entities.Entity.
// Если парсинг не удался, то возвращает ошибку.
func EntitiesFromDTO(entitiesData []api.EntityDTO, engineAPI engine.EnginePort) (map[string]entities.Entity, error) {
	IDToEntity := make(map[string]entities.Entity)

	for _, entityDTO := range entitiesData {
		entityType := strings.Split(entityDTO.ID, "_")[0]
		switch entityType {
		case entities.TypeLamp:
			lamp, err := devices.NewLamp(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}
			IDToEntity[entityDTO.ID] = lamp
		case entities.TypeLampSwitcher:
			lampSwitcher, err := devices.NewLampSwitcher(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}
			IDToEntity[entityDTO.ID] = lampSwitcher
		default:
			return nil, errors.New("cannot parse input data, invalid format")
		}
	}

	return IDToEntity, nil
}

// FieldFromDTO парсит данные о плане и возвращает config.FieldDTO, ошибку если данные некорректные.
func FieldFromDTO(fieldDTO api.FieldDTO) *field.Field {
	fieldCells := make([][]*field.Cell, fieldDTO.Height)

	for i, cells := range fieldDTO.Cells {
		fieldCells[i] = make([]*field.Cell, fieldDTO.Width)
		for j, cell := range cells {
			fieldCells[i][j] = &field.Cell{
				Condition:    cell.Condition,
				IsHiddenWall: false,
			}
		}
	}

	simField := field.NewField(fieldDTO.Width, fieldDTO.Height, fieldCells)

	return simField
}
