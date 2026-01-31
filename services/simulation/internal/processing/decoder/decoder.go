package decoder

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/entities"
)

//Декодирует все полученные данные и возвращает ошибку, если данные невалидный.
//
//- Возвращает в engine мапу EntityID -> Entity (значение = конкртеная структура, которая реализует интерфейс Entity.  Парсит на основе ID + EntityType)
//
//- Возвращает в engine мапу связей (EntityID -> []EntityID), кто кого тригерит
//
//- Возвращает структуру апдейта симуляции (ID, место, ...)

// ParseEntities парсит данные о сущностях и возвращает map[string]*entities.BaseEntity
// для сущностей без процессов и map[string]entities.EntityWithProces для
// сущностей с процессами. Если парсинг не удался, то возвращает ошибку.
func ParseEntities(data []config.EntityDTO) (map[string]entities.BaseEntity, map[string]entities.EntityWithProcess, error) {
	// TODO: switch по типу и создание конкретной структуры девайса
	panic("todo")
}

// ParseEvents парсит данные о событиях и возвращает []config.EventDTO или ошибку, если парсинг не удался.
func ParseEvents(data []byte) ([]config.EventDTO, error) {
	// TODO: switch по типу и создание конкретной структуры девайса
	panic("todo")
}

// ParseField парсит данные о плане и возвращает config.FieldDTO, ошибку если данные некорректные.
func ParseField(data []byte) (config.FieldDTO, error) {
	// TODO: switch по типу и создание конкретной структуры девайса
	panic("todo")
}
