-- +goose Up
-- Free-text names for people on a ticket visit who are not in the support-engineer list.
ALTER TABLE service_visits
  ADD COLUMN IF NOT EXISTS other_engineers VARCHAR(200);

-- +goose Down
ALTER TABLE service_visits DROP COLUMN IF EXISTS other_engineers;
