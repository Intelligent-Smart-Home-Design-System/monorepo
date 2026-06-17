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
		entityType := normalizeEntityType(entityDTO)
		switch entityType {
		case entities.TypeLamp:
			lamp, err := devices.NewLamp(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = lamp
		case entities.TypeSmartLamp:
			smartLamp, err := devices.NewSmartLamp(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = smartLamp
		case entities.TypeSwitcher:
			Switcher, err := devices.NewSwitcher(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = Switcher
		case entities.TypeSensorWithUpdate:
			sensor, err := devices.NewSensorWithUpdate(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = sensor
		case entities.TypeSensorWithIntStatus:
			sensor, err := devices.NewSensorWithIntStatus(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = sensor
		case entities.TypeRadiusMoveSensorWithoutUpdate:
			sensor, err := devices.NewRadiusSensorWithoutUpdate(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = sensor
		case entities.TypeRadiusMoveSensorWithUpdate:
			sensor, err := devices.NewRadiusSensorWithUpdate(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = sensor
		case entities.TypeHuman:
			human, err := actors.NewHuman(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = human
		case entities.TypeFire:
			fire, err := actors.NewFire(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = fire
		case entities.TypeSmartDimmer:
			dimmer, err := devices.NewSmartDimmer(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = dimmer
		case entities.TypeSensorWithoutUpdate:
			sensor, err := devices.NewSensorWithoutUpdate(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = sensor
		case entities.TypeSiren:
			siren, err := devices.NewSiren(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = siren
		case entities.TypeWindow:
			window, err := devices.NewWindow(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = window
		case entities.TypeDoor:
			door, err := devices.NewDoor(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = door
		case entities.TypeSmartLock:
			lock, err := devices.NewSmartLock(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = lock
		case entities.TypeSmartDoorbell:
			doorbell, err := devices.NewSmartDoorbell(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = doorbell
		case entities.TypeSmartCurtains:
			curtains, err := devices.NewSmartCurtains(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = curtains
		case entities.TypeCamera:
			camera, err := devices.NewCamera(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = camera
		case entities.TypeAirConditioner:
			airConditioner, err := devices.NewAirConditioner(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = airConditioner
		case entities.TypeThermostat:
			thermostat, err := devices.NewThermostat(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = thermostat
		case entities.TypeSmartFloor:
			smartFloor, err := devices.NewSmartFloor(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = smartFloor
		case entities.TypeTV:
			tv, err := devices.NewTV(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = tv
		case entities.TypeSubwoofer:
			subwoofer, err := devices.NewSubwoofer(entityDTO.Info, engineAPI)
			if err != nil {
				return nil, err
			}

			IDToEntity[entityDTO.ID] = subwoofer
		default:
			return nil, ErrorInvalidFormat
		}
	}

	return IDToEntity, nil
}

func normalizeEntityType(entityDTO api.EntityDTO) string {
	entityType := entityDTO.Type
	if entityType == "" {
		entityType = strings.Split(entityDTO.ID, "_")[0]
	}

	switch entityType {
	case "lamp_switcher", "lampSwitcher":
		return entities.TypeSwitcher
	case "motion_sensor", "presence_sensor":
		return entities.TypeSensorWithUpdate
	case "illumination_sensor":
		return entities.TypeSensorWithIntStatus
	case "door_sensor", "window_sensor", "wireless_button_switch":
		return entities.TypeSensorWithoutUpdate
	case "smart_bulb":
		if strings.HasPrefix(entityDTO.ID, "smartLamp") {
			return entities.TypeSmartLamp
		}
		return entities.TypeLamp
	case "smart_lamp":
		return entities.TypeSmartLamp
	case "smart_dimmer":
		return entities.TypeSmartDimmer
	case "smart_siren":
		return entities.TypeSiren
	case "smart_lock":
		return entities.TypeSmartLock
	case "smart_doorbell":
		return entities.TypeSmartDoorbell
	case "curtains":
		return entities.TypeSmartCurtains
	case "lamp_with":
		return entities.TypeRadiusMoveSensorWithoutUpdate
	case "sensor_with_update":
		return entities.TypeSensorWithUpdate
	case "sensor_without_update":
		return entities.TypeSensorWithoutUpdate
	case "sensor_with_int_status":
		return entities.TypeSensorWithIntStatus
	case "radius_move_sensor_with_update":
		return entities.TypeRadiusMoveSensorWithUpdate
	case "radius_move_sensor_without_update":
		return entities.TypeRadiusMoveSensorWithoutUpdate
	default:
		return entityType
	}
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
