// Package example — шаблон парсера/скрейпера. Методы: panic("not implemented").
//
// Все публичные методы намеренно вызывают panic("not implemented").
// Скопируйте internal/scrapers/example и internal/parsers/example, переименуйте пакет и Source.
//
// # Как применяются секции [example] в TOML (что реально в проекте сегодня)
//
// Pattern A — готовые URL в tracked_pages (DNS), но два разных канала:
//
//   A1. discovery_seeds — корень каталога для BFS (один или несколько URL).
//       scrape --discovery + seed → RunDiscoveryBFS(seeds) → CreateTask(category) по дереву.
//       Жёсткий вход «умная техника» = именно это; обход идёт от seed, не от search.
//
//   A2. search_queries — (опционально, отдельно от BFS) текст → URL поиска /search/?q=…
//       bootstrap только CreateTask("discovery", searchURL); BFS эти URL не читает.
//       Альтернативный вход «найти zigbee через поиск», не обход каталога.
//       Сейчас parse --discovery для DNS обрабатывает только category snapshots, не discovery
//       с поиска — канал A2 в пайплайне не доведён до listing (в отличие от BFS→category→parse).
//
//   В config.local.toml search_queries = [] — используется только A1.
//
// Pattern B — discovery_text_queries + discovery_url_template (как WB):
//   Реально: bootstrap кладёт wildberries://discovery/{query}; scraper строит API URL из шаблона.
//
// Секция «без discovery» (category_urls, listing_urls):
//   В продакшене сейчас:
//     - WB: одна category_url → CreateTask("category") на каждом scrape БЕЗ --discovery (scrape.go).
//     - Sprut: нет bootstrap — URL в tracked_pages вручную/SQL.
//     - listing_urls, category_urls (массив): только в ExampleConfig + BootstrapRegularTasks (заглушка).
//   Логика example: category_urls — массив (несколько фиксированных category); WB пока одна строка —
//   при реализации нового source предпочтительнее []string.
//
//   Поток с category_urls:
//     scrape (без --discovery) → bootstrap CreateTask(category) → scrape --page-types category
//     → parse → CreateTask(listing) → scrape listing → parse listing.
//
//   Поток с listing_urls (минимальный, как Sprut):
//     scrape → bootstrap CreateTask(listing) → scrape --page-types listing → parse.
//
// # Фильтры [jobs.*.<source>]
//
//   tracked_page_ids — (опционально) scrape и parse: tracked_pages.id.
//     На parse попадают все необработанные снимки этой страницы (их может быть несколько).
//
//   page_snapshot_ids — (опционально) только parse: page_snapshots.id.
//     Когда перескрапили страницу и нужен конкретный снимок, не все версии.
//
//   url_contains, limit, discovery_bootstrap — опционально; см. config/job_filter.go.
//
//   scraped_after / scraped_before — (опционально) parse: page_snapshots.scraped_at;
//     scrape: tracked_pages.last_scraped_at («когда последний раз ходили за HTML»).
//   created_after / created_before — (опционально) только scrape: tracked_pages.first_seen_at
//     («когда задача появилась» — BFS, parse --discovery, bootstrap). Для «только сегодняшние category».
//
// Какая колонка когда:
//   | Вопрос | Таблица | Колонка | Фильтр TOML |
//   | Задача создана сегодня (BFS/parse) | tracked_pages | first_seen_at | created_after |
//   | Скрапили сегодня | tracked_pages | last_scraped_at | scraped_after (scrape job) |
//   | Снимок сделан сегодня | page_snapshots | scraped_at | scraped_after (parse job) |
//   | Успешный scrape | tracked_pages | last_successful_scrape_at | (пока не в фильтрах) |
//
// # listingsOnly в ParseCategorySnapshots (DNS, опциональный паттерн)
//
// Один парсер category, два режима CLI:
//   listingsOnly=true  — parse --discovery: только listing tasks из BrowseLinks.ListingURLs.
//   listingsOnly=false — parse category: ещё PaginationURLs → category tasks.
// Если пагинации нет (WB) или отдельные job — флаг не нужен.
//
// # Опциональные компоненты по page_type
//
//   Scraper.Scrape              — обязательно
//   ListingParser               — если есть listing
//   DiscoveryParser             — (опционально) Pattern B, snapshot → listing URL
//   CategoryParser              — (опционально) snapshot category → listing URL
//   RunDiscoveryBFS             — (опционально) Pattern A, HTML-каталог с hub-страницами
//   browse.go / BrowseLinks     — (опционально) если category отдаёт listing + pagination + hubs
//   BootstrapDiscoverySeeds     — (опционально) при scrape --discovery
//   BootstrapRegularTasks       — (опционально) при обычном scrape
//   ParseCategorySnapshots        — (опционально) если category parse не вписывается в generic Worker
//
// # Пример TOML
//
//	[example]
//	discovery_seeds = ["https://shop.example/catalog/"]              # опционально, Pattern A
//	discovery_text_queries = ["умный дом"]                           # опционально, Pattern B
//	discovery_url_template = "https://api.example/search?q={query}&page={page}"
//	search_queries = ["zigbee"]                                        # опционально
//	search_url_template = "https://shop.example/search?q={query}&page={page}"
//	max_pages = 5
//	category_urls = ["https://shop.example/catalog/smart-home/"]     # опционально, без discovery
//	listing_urls = ["https://shop.example/product/1/"]               # опционально, без discovery
//	brand_aliases = { "яндекс" = "yandex" }
//
//	[jobs.parse.example]
//	limit = 10
//	scraped_after = 2026-06-01T00:00:00+03:00
//	# scraped_before не задан = до «сейчас»
//	page_snapshot_ids = [9001]
//	tracked_page_ids = [42]
//	url_contains = ["sensor"]
package example
