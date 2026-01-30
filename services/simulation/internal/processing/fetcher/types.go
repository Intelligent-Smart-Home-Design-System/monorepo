package fetcher

// Fetcher опреляет интрфейс для получения данных
type Fetcher interface {
	GetUpdates() map[string]any
}
