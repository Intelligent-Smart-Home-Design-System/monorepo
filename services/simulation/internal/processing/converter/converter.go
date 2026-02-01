package converter

import (
	"errors"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/devices"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/field"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/api"
)

// EntitiesFromDTO парсит данные о сущностях и возвращает map[string]*entities.Entity.
// Если парсинг не удался, то возвращает ошибку.
func EntitiesFromDTO(data []config.EntityDTO, engineAPI api.EngineAPI) (map[string]entities.Entity, error) {
	IDToEntity := make(map[string]entities.Entity)

	for _, entityDTO := range data {
		switch entityDTO.Type {
		case entities.TypeLamp:
			lamp, err := devices.NewLamp(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}
			IDToEntity[entityDTO.ID] = lamp
		default:
			return nil, errors.New("cannot parse input data, invalid format")
		}
	}

	return IDToEntity, nil
}

// FieldFromDTO парсит данные о плане и возвращает config.FieldDTO, ошибку если данные некорректные.
func FieldFromDTO(fieldDTO config.FieldDTO) *field.Field {
	fieldCells := make([][]*field.Cell, fieldDTO.Height)

	for i, cells := range fieldDTO.Cells {
		fieldCells[i] = make([]*field.Cell, fieldDTO.Width)
		for j, cell := range cells {
			fieldCells[i][j] = &field.Cell{
				X:            cell.X,
				Y:            cell.Y,
				Condition:    0,
				IsHiddenWall: false,
			}
		}
	}

	simField := field.NewField(fieldDTO.Width, fieldDTO.Height, fieldCells)

	return simField
}
