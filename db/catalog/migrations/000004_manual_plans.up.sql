ALTER TABLE frontend_plans
    ADD COLUMN IF NOT EXISTS floor_plan JSONB;
