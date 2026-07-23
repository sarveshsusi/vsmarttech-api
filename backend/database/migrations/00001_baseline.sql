-- +goose Up
-- Historical baseline: production schema was created by GORM AutoMigrate.
-- This version records that baseline so forward goose migrations can run safely.
-- On empty databases, the API still runs AutoMigrate once before goose (see database.Migrate).
SELECT 1;

-- +goose Down
SELECT 1;
