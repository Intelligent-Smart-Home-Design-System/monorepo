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
  ('wildberries', 'discovery', 'wildberries://discovery/умная лампочка', true),
  ('wildberries', 'discovery', 'wildberries://discovery/умный датчик протечки воды', true),
  ('wildberries', 'discovery', 'wildberries://discovery/умная колонка алиса', true)
ON CONFLICT (url_hash) DO NOTHING;

-- ── Category (опционально: scraper-scrape без --discovery) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('wildberries', 'category', 'https://www.wildberries.ru/catalog/0/search.aspx?search=%D1%83%D0%BC%D0%BD%D0%B0%D1%8F+%D0%BA%D0%BE%D0%BB%D0%BE%D0%BD%D0%BA%D0%B0+%D0%B0%D0%BB%D0%B8%D1%81%D0%B0', true)
ON CONFLICT (url_hash) DO NOTHING;

-- ── Listing (карточки товаров; обычно появляются после parse-discovery, seed — для прямого теста) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('wildberries', 'listing', 'https://www.wildberries.ru/catalog/443786302/detail.aspx', true),
  ('wildberries', 'listing', 'https://www.wildberries.ru/catalog/234709222/detail.aspx', true)
ON CONFLICT (url_hash) DO NOTHING;

-- ── Compatibility (Yandex Zigbee — scraper-scrape / scraper-parse) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('yandex', 'compatibility', 'https://alice.yandex.ru/support/ru/smart-home/supported-zigbee-devices', true)
ON CONFLICT (url_hash) DO NOTHING;

-- ── Sprut listings (reference catalog, no prices) ──

INSERT INTO tracked_pages (source_name, page_type, url, is_active) VALUES
  ('sprut', 'listing', 'https://sprut.ai/catalog/item/aqara-temperature-and-humidity-sensor-t1', true),
  ('sprut', 'listing', 'https://sprut.ai/catalog/item/yandeks-stanciya-2', true),
  ('sprut', 'listing', 'https://sprut.ai/catalog/item/danalock-v3-smart-lock-zigbee', true)
ON CONFLICT (url_hash) DO NOTHING;
