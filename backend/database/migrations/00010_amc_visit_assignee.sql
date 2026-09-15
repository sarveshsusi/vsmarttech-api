-- +goose Up
-- Stamp each AMC visit with the assigned engineer so pending visits keep
-- a visible assignee, including after the AMC is reassigned.
ALTER TABLE amc_visits
  ADD COLUMN IF NOT EXISTS support_engineer_id UUID;

UPDATE amc_visits v
SET support_engineer_id = a.support_engineer_id
FROM amc_assignments a
WHERE v.amc_assignment_id = a.id
  AND v.support_engineer_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_amc_visits_support_engineer_id
  ON amc_visits (support_engineer_id);

-- +goose Down
DROP INDEX IF EXISTS idx_amc_visits_support_engineer_id;
ALTER TABLE amc_visits DROP COLUMN IF EXISTS support_engineer_id;
