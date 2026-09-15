-- +goose Up
CREATE TABLE IF NOT EXISTS amc_assignment_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  amc_assignment_id UUID NOT NULL REFERENCES amc_assignments(id) ON DELETE CASCADE,
  event_type VARCHAR(32) NOT NULL,
  actor_user_id UUID NOT NULL,
  from_engineer_id UUID,
  to_engineer_id UUID NOT NULL,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_amc_assignment_events_assignment
  ON amc_assignment_events (amc_assignment_id, created_at);

CREATE INDEX IF NOT EXISTS idx_amc_assignment_events_type
  ON amc_assignment_events (event_type);

INSERT INTO amc_assignment_events (
  amc_assignment_id,
  event_type,
  actor_user_id,
  to_engineer_id,
  created_at
)
SELECT
  a.id,
  'assigned',
  a.assigned_by,
  a.support_engineer_id,
  COALESCE(a.assigned_at, a.created_at, NOW())
FROM amc_assignments a
WHERE a.assigned_by IS NOT NULL
  AND a.support_engineer_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM amc_assignment_events e
    WHERE e.amc_assignment_id = a.id
  );

-- +goose Down
DROP INDEX IF EXISTS idx_amc_assignment_events_type;
DROP INDEX IF EXISTS idx_amc_assignment_events_assignment;
DROP TABLE IF EXISTS amc_assignment_events;
