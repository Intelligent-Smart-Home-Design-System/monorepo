CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Bronze layer

CREATE TABLE tracked_pages (
    id SERIAL PRIMARY KEY,
    source_name VARCHAR(100) NOT NULL,  -- 'amazon_us', 'sprut_ai', 'wildberries', 'yandex'
    page_type TEXT NOT NULL, -- 'listing', 'compatibility', 'cloud_integration', ...
    url TEXT NOT NULL,
    url_hash VARCHAR(64) GENERATED ALWAYS AS (encode(digest(url::bytea, 'sha256'), 'hex')) STORED,

    -- Status tracking (для мониторинга)
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_scraped_at TIMESTAMP,
    last_successful_scrape_at TIMESTAMP,
    scrape_count INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,  -- Set false if 404'd multiple times
    
    UNIQUE(url_hash)
);

CREATE TABLE page_snapshots (
    id SERIAL PRIMARY KEY,
    tracked_page INTEGER REFERENCES tracked_pages(id),
    scraped_at TIMESTAMP DEFAULT NOW(),
    processed BOOLEAN DEFAULT FALSE,
    warc_bundle_archive BYTEA, -- .tar.gz of all .warc of json/html downloaded with page
    scrape_duration_ms INTEGER
);

-- Silver layer

CREATE TABLE parsed_listing_snapshots (
    id SERIAL PRIMARY KEY,
    page_snapshot_id INTEGER REFERENCES page_snapshots(id),
    parsed_at TIMESTAMP DEFAULT NOW(),
    processed BOOLEAN DEFAULT FALSE,

    extracted_in_stock BOOLEAN NOT NULL, -- есть в продаже
    extracted_text TEXT NOT NULL,  -- Здесь весь релевантный текст с описанием товара (описание, общие характеристики, ...)

    extracted_name TEXT NOT NULL,
    extracted_brand TEXT NOT NULL,
    extracted_image_url TEXT,
    extracted_price INTEGER,
    extracted_currency VARCHAR(3),  -- 'RUB', 'USD', 'EUR'
    extracted_model_number TEXT,
    extracted_category TEXT, -- 'water_leak_detector', 'smart_lamp', ...
    
    -- Количество устройств в комплекте
    extracted_quantity INTEGER,  -- 1, 2, 3, 4...
    extracted_quantity_raw TEXT,  -- "3-Pack", "Set of 2"
    
    -- Ratings on page
    extracted_rating NUMERIC(3,2) NOT NULL,  -- 4.5
    extracted_review_count INTEGER NOT NULL,  -- 1,234
    
    extractor_version VARCHAR(20),  -- Track which version extracted this
    
    -- Content hash for detecting changes
    content_hash VARCHAR(64) -- хэш всех полей extracted_. Если ничего не поменялось, меняем только parsed_at в последнем снепшоте
);

CREATE TABLE yandex_cloud_integrations (
    id SERIAL PRIMARY KEY,
    
    -- Bridging
    -- meaning: devices added to ecosystem_source can be exported to yandex home
    ecosystem_source TEXT NOT NULL,
    
    -- text describing the integration - may contain model numbers, or series, or other fuzzy selectors like 'Умные розетки'
    description TEXT NOT NULL,

    -- Where we learned this (nullable = rule-based or manual)
    tracked_page_id INTEGER REFERENCES tracked_pages(id),
    
    -- Timestamps
    discovered_at TIMESTAMP DEFAULT NOW(),
    last_confirmed_at TIMESTAMP,        -- updated when we re-scrape source and it's still there
    
    UNIQUE(ecosystem_source)
);

-- Gold layer

CREATE TABLE llm_extracted_listings (
    id SERIAL PRIMARY KEY,
    parsed_listing_snapshot_id INTEGER REFERENCES parsed_listing_snapshots(id),
    extracted_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- identification
    brand TEXT NOT NULL,           -- "Яндекс" (cleaned up)
    model TEXT NOT NULL,           -- "YNDX-00558, or e27:8lm, ..."
    
    -- Classification
    category TEXT NOT NULL,        -- 'smart_lamp', 'motion_sensor', 'hub'
    category_confidence FLOAT,

    -- Category-specific attributes
    device_attributes JSONB NOT NULL,
    
    -- taxonomy version (git tag), e.g. v1, v2, ...
    taxonomy_version TEXT NOT NULL,

    -- about llm extraction
    llm_model TEXT NOT NULL                  -- 'gpt-4o', 'claude-sonnet'
);

CREATE TABLE device (
    id SERIAL PRIMARY KEY,
    
    -- identification
    brand TEXT NOT NULL,           -- "Яндекс" (cleaned up)
    model TEXT NOT NULL,           -- "YNDX-00558, or e27:8lm, ..."
    
    -- Classification (assume category is one with highest confidence?)
    category TEXT NOT NULL,        -- 'smart_lamp', 'motion_sensor', 'hub'

    -- Category-specific attributes(merged from multiple llm extracted ones)
    device_attributes JSONB NOT NULL,

    -- taxonomy version (git tag), e.g. v1, v2, ...
    taxonomy_version TEXT NOT NULL
);

CREATE TABLE listing_device_links (
    llm_extracted_listing_id INTEGER PRIMARY KEY REFERENCES llm_extracted_listings(id),
    device_id INTEGER REFERENCES device(id),

    linked_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE direct_compatibility (
    id SERIAL PRIMARY KEY,
    
    -- What device
    brand TEXT NOT NULL,
    model TEXT NOT NULL,
    
    -- Compatible with what
    ecosystem TEXT NOT NULL,
    
    -- Where we learned this (nullable = rule-based or manual)
    tracked_page_id INTEGER REFERENCES tracked_pages(id),
    
    -- Timestamps
    discovered_at TIMESTAMP DEFAULT NOW(),
    last_confirmed_at TIMESTAMP,        -- updated when we re-scrape source and it's still there
    
    UNIQUE(brand, model, ecosystem)
);

CREATE TABLE bridge_ecosystem_compatibility (
    id SERIAL PRIMARY KEY,
    
    -- What device
    brand TEXT NOT NULL,
    model TEXT NOT NULL,
    
    -- Bridging
    -- meaning: device added to ecosystem_source can be exported to ecosystem_target
    -- example: source=aqara target=apple protocol=matter
    ecosystem_source TEXT NOT NULL,
    ecosystem_target TEXT NOT NULL,
    protocol TEXT NOT NULL, -- over which protocol
    
    -- Where we learned this (nullable = rule-based or manual)
    tracked_page_id INTEGER REFERENCES tracked_pages(id),
    
    -- Timestamps
    discovered_at TIMESTAMP DEFAULT NOW(),
    last_confirmed_at TIMESTAMP,        -- updated when we re-scrape source and it's still there
    
    UNIQUE(brand, model, ecosystem_source, ecosystem_target, protocol)
);
