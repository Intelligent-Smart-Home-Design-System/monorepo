-- Начальные задачи scraper (таблица tracked_pages).
-- Первая стадия pipeline (scraper-scrape-discovery) читает активные строки из БД.
-- Подставьте свои URL/запросы и выполните: make pipeline-seed
--
-- Форматы url по page_type:
--   discovery     wildberries://discovery/<поисковый запрос WB>
--   listing       https://www.wildberries.ru/catalog/<id>/detail.aspx
--   category      https://www.wildberries.ru/catalog/...  (раздел каталога)
--   compatibility https://alice.yandex.ru/support/ru/smart-home/supported-zigbee-devices
--
-- source_name: wildberries | yandex | sprut | printer
-- Повторный запуск безопасен: дубликаты по url_hash игнорируются.

-- ── Discovery (первая job: scraper-scrape-discovery / scraper-parse-discovery) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('wildberries', 'discovery', 'wildberries://discovery/ЗАМЕНИТЕ_ПОИСКОВЫЙ_ЗАПРОС_1', true),
  ('wildberries', 'discovery', 'wildberries://discovery/ЗАМЕНИТЕ_ПОИСКОВЫЙ_ЗАПРОС_2', true),
  ('wildberries', 'discovery', 'wildberries://discovery/умная лампа яндекс', true)
ON CONFLICT (url_hash) DO NOTHING;

-- ── Category (опционально: scraper-scrape без --discovery) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('wildberries', 'category', 'https://www.wildberries.ru/catalog/elektronika/umnyy-dom', true)
ON CONFLICT (url_hash) DO NOTHING;

-- ── Listing (карточки товаров; обычно появляются после parse-discovery, seed — для прямого теста) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('wildberries', 'listing', 'https://www.wildberries.ru/catalog/ЗАМЕНИТЕ_ID_ТОВАРА/detail.aspx', true),
  ('wildberries', 'listing', 'https://www.wildberries.ru/catalog/ЗАМЕНИТЕ_ID_ТОВАРА_2/detail.aspx', true)
ON CONFLICT (url_hash) DO NOTHING;

-- ── Compatibility (Yandex Zigbee — scraper-scrape / scraper-parse) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('yandex', 'compatibility', 'https://alice.yandex.ru/support/ru/smart-home/supported-zigbee-devices', true)
ON CONFLICT (url_hash) DO NOTHING;
