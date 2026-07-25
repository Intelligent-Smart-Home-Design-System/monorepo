ALTER TABLE frontend_plans
    DROP COLUMN IF EXISTS dependencies,
    DROP COLUMN IF EXISTS layout;
