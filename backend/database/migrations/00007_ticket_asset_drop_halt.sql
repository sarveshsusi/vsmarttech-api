-- +goose Up
-- Halted ticket status + SLA pause fields + asset drop requests

ALTER TABLE tickets ADD COLUMN IF NOT EXISTS sla_paused_at TIMESTAMPTZ;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS sla_paused_total_seconds INT NOT NULL DEFAULT 0;

-- Replace status CHECK to include Halted (Postgres cannot ALTER CHECK in place)
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_status_check;
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'tickets_status_check'
  ) THEN
    ALTER TABLE tickets DROP CONSTRAINT tickets_status_check;
  END IF;
EXCEPTION WHEN undefined_object THEN
  NULL;
END $$;

-- Drop any leftover check on status (name varies by how it was created)
DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN
    SELECT c.conname
    FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE t.relname = 'tickets'
      AND c.contype = 'c'
      AND pg_get_constraintdef(c.oid) ILIKE '%status%'
  LOOP
    EXECUTE format('ALTER TABLE tickets DROP CONSTRAINT IF EXISTS %I', r.conname);
  END LOOP;
END $$;

ALTER TABLE tickets
  ADD CONSTRAINT tickets_status_check
  CHECK (status IN ('Open', 'Assigned', 'In Progress', 'Halted', 'Closed'));

CREATE TABLE IF NOT EXISTS ticket_asset_drops (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ticket_id VARCHAR(20) NOT NULL REFERENCES tickets(id),
  serial_number VARCHAR(120) NOT NULL,
  name VARCHAR(200) NOT NULL,
  model VARCHAR(120) NOT NULL DEFAULT '',
  category VARCHAR(80) NOT NULL DEFAULT '',
  site_location VARCHAR(255) NOT NULL DEFAULT '',
  is_replacement BOOLEAN NOT NULL DEFAULT FALSE,
  status VARCHAR(32) NOT NULL DEFAULT 'requested'
    CHECK (status IN ('requested', 'acknowledged', 'return_assigned', 'returned')),
  asset_id UUID REFERENCES assets(id),
  return_engineer_id UUID REFERENCES support_engineers(id),
  requested_by UUID NOT NULL REFERENCES users(id),
  acknowledged_by UUID REFERENCES users(id),
  acknowledged_at TIMESTAMPTZ,
  return_assigned_at TIMESTAMPTZ,
  returned_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ticket_asset_drops_ticket_id ON ticket_asset_drops (ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_asset_drops_status ON ticket_asset_drops (status);
CREATE INDEX IF NOT EXISTS idx_ticket_asset_drops_return_engineer_id ON ticket_asset_drops (return_engineer_id);
CREATE INDEX IF NOT EXISTS idx_ticket_asset_drops_asset_id ON ticket_asset_drops (asset_id);

-- +goose Down
DROP TABLE IF EXISTS ticket_asset_drops;

ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_status_check;
ALTER TABLE tickets
  ADD CONSTRAINT tickets_status_check
  CHECK (status IN ('Open', 'Assigned', 'In Progress', 'Closed'));

ALTER TABLE tickets DROP COLUMN IF EXISTS sla_paused_at;
ALTER TABLE tickets DROP COLUMN IF EXISTS sla_paused_total_seconds;
