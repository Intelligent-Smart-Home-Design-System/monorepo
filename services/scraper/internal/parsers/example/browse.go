package example

// BrowseLinks — результат разбора catalog/category страницы (паттерн DNS).
//
// Используйте, если одна HTML-страница содержит несколько типов ссылок.
// Альтернатива — возвращать []string только с listing URL (паттерн Wildberries CategoryParser).
type BrowseLinks struct {
	// DiscoveryURLs — hub/подкатегории → CreateTask(page_type=discovery) или очередь BFS в памяти.
	DiscoveryURLs []string
	// ListingURLs — карточки товаров → CreateTask(page_type=listing).
	ListingURLs []string
	// PaginationURLs — следующие страницы сетки → CreateTask(page_type=category).
	PaginationURLs []string
}

// IsProductGrid сообщает, что страница — листинг товаров (а не hub категорий).
// В BFS: hub → только enqueue в память; grid → CreateTask(category).
func (l *BrowseLinks) IsProductGrid() bool {
	panic("not implemented")
}

// ExtractBrowseLinks разбирает HTML hub- или category-страницы.
//
// Вызывается из DiscoveryParser, CategoryParser или RunDiscoveryBFS/processPage.
// pageURL нужен для разрешения относительных ссылок и пагинации.
//
// Верните ошибку, если на странице нет ожидаемых ссылок — snapshot будет помечен processed с ошибкой в логе.
func ExtractBrowseLinks(html []byte, pageURL string) (*BrowseLinks, error) {
	panic("not implemented")
}
