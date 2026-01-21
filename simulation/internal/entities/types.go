package entities

// Entity определяет главный интерфейс устройств
type Entity interface {
	// Trigger возвращает время в секундах, с начала симуляции
	Trigger() float64

	// SetEvent создает новый event и обновляет соответствующее поле
	SetEvent()

	// SetEvent создает новый event и обновляет соответствующее поле
	SetDelay()

	//GetID() string
	//GetType() string
	//GetReactionDelay() time.Duration
}
