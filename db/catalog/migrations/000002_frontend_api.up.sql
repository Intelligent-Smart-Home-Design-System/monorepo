CREATE TABLE frontend_device_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    filters JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE frontend_ecosystems (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    may_be_main BOOLEAN NOT NULL DEFAULT TRUE,
    image_url TEXT
);

CREATE TABLE frontend_presets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    requirements JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE frontend_plans (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    budget DOUBLE PRECISION NOT NULL,
    main_ecosystem_id TEXT NOT NULL,
    allowed_ecosystems TEXT[],
    excluded_ecosystems TEXT[],
    requirements JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'completed' CHECK (status IN ('queued', 'generating', 'completed', 'failed')),
    progress DOUBLE PRECISION,
    error JSONB,
    bundles JSONB NOT NULL DEFAULT '[]'::jsonb
);

INSERT INTO frontend_device_types (id, name, filters) VALUES
    ('water_leak_sensor', 'Water Leak Sensor', '[{"name":"Probe type","field":"probe_type","value_type":"string","enum_values":["contact"],"operations":["eq","neq"]},{"name":"Battery life, years","field":"battery_life_years","value_type":"number","enum_values":null,"operations":["gte","lte","gt","lt"]}]'::jsonb),
    ('gas_leak_sensor', 'Gas Leak Sensor', '[{"name":"Gas type","field":"gas_types","value_type":"string","enum_values":["methane","natural_gas"],"operations":["contains","exists"]},{"name":"Battery life, years","field":"battery_life_years","value_type":"number","enum_values":null,"operations":["gte","lte","gt","lt"]}]'::jsonb),
    ('smart_lamp', 'Smart Lamp', '[{"name":"Socket type","field":"socket_type","value_type":"string","enum_values":["E27","E14"],"operations":["eq","neq"]}]'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    filters = EXCLUDED.filters;

INSERT INTO frontend_ecosystems (id, name, description, may_be_main, image_url) VALUES
    ('yandex', 'Yandex Home', 'Smart home ecosystem and voice assistant platform.', TRUE, NULL),
    ('aqara', 'Aqara', 'Zigbee-oriented smart home ecosystem.', TRUE, NULL),
    ('google', 'Google Home', 'Google smart home ecosystem.', TRUE, NULL),
    ('tuya', 'Tuya Smart', 'Tuya cloud smart home ecosystem.', TRUE, NULL)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    may_be_main = EXCLUDED.may_be_main,
    image_url = EXCLUDED.image_url;

INSERT INTO frontend_presets (id, name, description, requirements) VALUES
    ('security-basic', 'Basic Security', 'Leak and gas safety baseline.', '[{"id":1,"device_type":"water_leak_sensor","quantity":1,"filters":[]},{"id":2,"device_type":"gas_leak_sensor","quantity":1,"filters":[]}]'::jsonb),
    ('lighting-basic', 'Basic Lighting', 'Simple smart lighting setup.', '[{"id":1,"device_type":"smart_lamp","quantity":2,"filters":[]}]'::jsonb),
    ('comfort', 'Comfort', 'Lighting plus basic safety devices.', '[{"id":1,"device_type":"smart_lamp","quantity":2,"filters":[]},{"id":2,"device_type":"water_leak_sensor","quantity":1,"filters":[]},{"id":3,"device_type":"gas_leak_sensor","quantity":1,"filters":[]}]'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    requirements = EXCLUDED.requirements;
