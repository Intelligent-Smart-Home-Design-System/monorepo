package converter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/actors"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities/devices"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/processing/engine"
)

var (
	ErrorInvalidFormat = errors.New("cannot parse input data, invalid format")
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
		case entities.TypeLightSwitchOffSensor:
			switcher, err := devices.NewLightSwitchOffSensor(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}
			IDToEntity[entityDTO.ID] = switcher
		case entities.TypeHuman:
			human, err := actors.NewHuman(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}
			IDToEntity[entityDTO.ID] = human
		default:
			return nil, ErrorInvalidFormat
		}
	}

	return IDToEntity, nil
}

// ParseFloor парсит данные о плане.
func ParseFloor(data []byte) (*api.Floor, error) {
	var floor api.Floor
	if err := json.Unmarshal(data, &floor); err != nil {
		return nil, fmt.Errorf("unmarshal floor json: %w", err)
	}

	floor.Adjacency = make(map[string][]api.RoomEdge)
	for i := range floor.Doors {
		door := &floor.Doors[i]
		if len(door.Rooms) != 2 {
			continue
		}
		aID, bID := door.Rooms[0], door.Rooms[1]
		floor.Adjacency[aID] = append(floor.Adjacency[aID], api.RoomEdge{
			NeighborRoomID: bID,
			Door:           door,
		})
		floor.Adjacency[bID] = append(floor.Adjacency[bID], api.RoomEdge{
			NeighborRoomID: aID,
			Door:           door,
		})
	}

	return &floor, nil
}

// DependenciesFromDTO парсит данные о зависимостях
func DependenciesFromDTO(scenarios []api.ScenarioDTO) map[string][]api.EdgeDTO {
	IDToDependencies := make(map[string][]api.EdgeDTO)

	for _, scenario := range scenarios {
		for _, edge := range scenario.Edges {
			IDToDependencies[scenario.EntityID] = append(IDToDependencies[scenario.EntityID], api.EdgeDTO{
				ToID:   edge.ToID,
				Action: edge.Action,
			})
		}
	}

	return IDToDependencies
}
