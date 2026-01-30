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

type Decoder struct {
}

// ParseEntities парсит данные о сущностях и возвращает map[string]*entities.Entity или ошибку, если парсинг не удался.
func (d *Decoder) ParseEntities(data []config.EntityDTO) (map[string]*entities.Entity, error) {
	// TODO: switch по типу и создание конкретной структуры девайса
	panic("todo")
}

// ParseEvents парсит данные о событиях и возвращает []config.EventDTO или ошибку, если парсинг не удался.
func (d *Decoder) ParseEvents(data []byte) ([]config.EventDTO, error) {
	// TODO: switch по типу и создание конкретной структуры девайса
	panic("todo")
}

// ParseField парсит данные о плане и возвращает config.FieldDTO, ошибку если данные некорректные.
func (d *Decoder) ParseField(data []byte) (config.FieldDTO, error) {
	// TODO: switch по типу и создание конкретной структуры девайса
	panic("todo")
}
