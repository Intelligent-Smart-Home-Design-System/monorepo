INSERT INTO devices (id, brand, model, category, quality, device_attributes, taxonomy_version) VALUES
  (1, 'Aqara', 'Leak Sensor', 'water_leak_sensor', 0.86, '{"probe_type":"contact","battery_life_years":2}'::jsonb, 'test'),
  (2, 'Aqara', 'Gas Sensor', 'gas_leak_sensor', 0.82, '{"gas_types":["methane"],"battery_life_years":2}'::jsonb, 'test'),
  (3, 'Yandex', 'Smart Bulb', 'smart_lamp', 0.80, '{"socket_type":"E27"}'::jsonb, 'test')
ON CONFLICT (id) DO NOTHING;

INSERT INTO parsed_listing_snapshots (
  id, extracted_in_stock, extracted_text, extracted_name, extracted_brand,
  extracted_price, extracted_currency, extracted_category, extracted_rating,
  extracted_review_count
) VALUES
  (1, true, 'Leak Sensor', 'Leak Sensor', 'Aqara', 1200, 'RUB', 'water_leak_sensor', 4.7, 120),
  (2, true, 'Gas Sensor', 'Gas Sensor', 'Aqara', 2200, 'RUB', 'gas_leak_sensor', 4.6, 80),
  (3, true, 'Smart Bulb', 'Smart Bulb', 'Yandex', 900, 'RUB', 'smart_lamp', 4.5, 200)
ON CONFLICT (id) DO NOTHING;

INSERT INTO llm_extracted_listings (
  id, parsed_listing_snapshot_id, brand, model, category,
  category_confidence, device_attributes, taxonomy_version, llm_model
) VALUES
  (1, 1, 'Aqara', 'Leak Sensor', 'water_leak_sensor', 1.0, '{}'::jsonb, 'test', 'seed'),
  (2, 2, 'Aqara', 'Gas Sensor', 'gas_leak_sensor', 1.0, '{}'::jsonb, 'test', 'seed'),
  (3, 3, 'Yandex', 'Smart Bulb', 'smart_lamp', 1.0, '{}'::jsonb, 'test', 'seed')
ON CONFLICT (id) DO NOTHING;

INSERT INTO listing_device_links (llm_extracted_listing_id, device_id) VALUES
  (1, 1),
  (2, 2),
  (3, 3)
ON CONFLICT (llm_extracted_listing_id) DO NOTHING;

INSERT INTO direct_compatibility (device_id, ecosystem, protocol) VALUES
  (1, 'yandex', 'zigbee'),
  (2, 'yandex', 'zigbee'),
  (3, 'yandex', 'wifi')
ON CONFLICT (device_id, ecosystem, protocol) DO NOTHING;
