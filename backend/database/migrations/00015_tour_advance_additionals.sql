-- +goose Up
CREATE TABLE IF NOT EXISTS tour_advance_additionals (
  id UUID PRIMARY KEY,
  tour_advance_request_id UUID NOT NULL REFERENCES tour_advance_requests(id) ON DELETE CASCADE,
  amount NUMERIC(12,2) NOT NULL,
  reason TEXT NOT NULL,
  status VARCHAR(32) NOT NULL,
  approved_amount NUMERIC(12,2),
  approval_remarks TEXT NOT NULL DEFAULT '',
  approved_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  approved_at TIMESTAMPTZ,
  rejection_remarks TEXT NOT NULL DEFAULT '',
  rejected_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  rejected_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_advance_additionals_request
  ON tour_advance_additionals (tour_advance_request_id);
CREATE INDEX IF NOT EXISTS idx_tour_advance_additionals_status
  ON tour_advance_additionals (status);

ALTER TABLE tour_advance_additionals DROP CONSTRAINT IF EXISTS tour_advance_additional_status_check;
ALTER TABLE tour_advance_additionals ADD CONSTRAINT tour_advance_additional_status_check CHECK (
  status IN ('PENDING_APPROVAL', 'APPROVED', 'REJECTED')
);

ALTER TABLE tour_advance_additionals DROP CONSTRAINT IF EXISTS tour_advance_additional_amount_check;
ALTER TABLE tour_advance_additionals ADD CONSTRAINT tour_advance_additional_amount_check CHECK (
  amount > 0
);

ALTER TABLE tour_advance_additionals DROP CONSTRAINT IF EXISTS tour_advance_additional_approved_check;
ALTER TABLE tour_advance_additionals ADD CONSTRAINT tour_advance_additional_approved_check CHECK (
  approved_amount IS NULL
  OR (approved_amount > 0 AND approved_amount <= amount)
);

-- +goose Down
DROP TABLE IF EXISTS tour_advance_additionals;
