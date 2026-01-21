package entities

// Entity определяет главный интерфейс устройств
type Entity interface {
	// Trigger возвращает время в секундах, с начала симуляции
	Trigger(delay int) float64

	// SetEvent создает новый event и обновляет соответствующее поле
	SetEvent()

	Process()

	//GetID() string
	//GetType() string
	//GetReactionDelay() time.Duration
}
