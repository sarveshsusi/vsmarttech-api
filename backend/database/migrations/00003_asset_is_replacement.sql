-- +goose Up
ALTER TABLE assets ADD COLUMN IF NOT EXISTS is_replacement BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE assets DROP COLUMN IF EXISTS is_replacement;
