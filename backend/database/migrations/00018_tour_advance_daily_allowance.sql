-- +goose Up
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS daily_allowance_rate NUMERIC(12,2) NOT NULL DEFAULT 0;
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS daily_allowance NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_allowance_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_allowance_check CHECK (
  daily_allowance_rate >= 0 AND daily_allowance >= 0
);

ALTER TABLE tour_advance_policies
  ADD COLUMN IF NOT EXISTS daily_allowance_rate NUMERIC(12,2);

-- +goose Down
ALTER TABLE tour_advance_policies DROP COLUMN IF EXISTS daily_allowance_rate;
ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_allowance_check;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS daily_allowance;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS daily_allowance_rate;
