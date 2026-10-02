-- +goose Up
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS travellers JSONB;
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS travel_from VARCHAR(10);
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS travel_to VARCHAR(10);

-- +goose Down
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS travel_to;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS travel_from;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS travellers;
