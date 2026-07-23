-- +goose Up
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS family_id UUID;

UPDATE refresh_tokens
SET family_id = id
WHERE family_id IS NULL
   OR family_id = '00000000-0000-0000-0000-000000000000';

ALTER TABLE refresh_tokens ALTER COLUMN family_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_id ON refresh_tokens (family_id);

-- +goose Down
DROP INDEX IF EXISTS idx_refresh_tokens_family_id;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS family_id;
