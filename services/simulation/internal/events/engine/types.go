package engine

import "time"

// Engine определяет главный интерфейс для запуска и обработки симуляции
type Engine interface {
	// SetStartTime устанавливает время начала первого события в simulation и возвращает его.
	// Далее узнаем текущее время из simulation
	SetStartTime(duration time.Duration)
}
