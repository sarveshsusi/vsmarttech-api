-- +goose Up
CREATE SEQUENCE IF NOT EXISTS tour_advance_request_number_seq START 1;

CREATE TABLE IF NOT EXISTS tour_advance_requests (
  id UUID PRIMARY KEY,
  request_number VARCHAR(16) NOT NULL,
  engineer_id UUID NOT NULL REFERENCES support_engineers(id) ON DELETE RESTRICT,
  customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
  location_snapshot VARCHAR(150) NOT NULL,
  work_type VARCHAR(20) NOT NULL,
  po_number VARCHAR(100),
  ticket_id VARCHAR(20) REFERENCES tickets(id) ON DELETE RESTRICT,
  persons_travelling INTEGER NOT NULL,
  days_planned INTEGER NOT NULL,
  food_expense NUMERIC(12,2) NOT NULL,
  local_expense NUMERIC(12,2) NOT NULL,
  accommodation_expense NUMERIC(12,2) NOT NULL,
  travel_mode VARCHAR(10) NOT NULL,
  travel_expense NUMERIC(12,2) NOT NULL,
  requested_amount NUMERIC(12,2) NOT NULL,
  approved_amount NUMERIC(12,2),
  status VARCHAR(32) NOT NULL,
  approval_remarks TEXT NOT NULL DEFAULT '',
  approved_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  approved_at TIMESTAMPTZ,
  rejection_remarks TEXT NOT NULL DEFAULT '',
  rejected_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  rejected_at TIMESTAMPTZ,
  bill_submitted_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  bill_submitted_at TIMESTAMPTZ,
  processed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_advance_requests_request_number
  ON tour_advance_requests (request_number);
CREATE INDEX IF NOT EXISTS idx_tour_advance_requests_engineer_id
  ON tour_advance_requests (engineer_id);
CREATE INDEX IF NOT EXISTS idx_tour_advance_requests_customer_id
  ON tour_advance_requests (customer_id);
CREATE INDEX IF NOT EXISTS idx_tour_advance_requests_status
  ON tour_advance_requests (status);
CREATE INDEX IF NOT EXISTS idx_tour_advance_requests_status_created
  ON tour_advance_requests (status, created_at);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_work_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_work_check CHECK (
  (
    work_type = 'INSTALLATION'
    AND po_number IS NOT NULL
    AND length(btrim(po_number)) > 0
    AND ticket_id IS NULL
  )
  OR
  (
    work_type = 'SERVICE'
    AND ticket_id IS NOT NULL
    AND po_number IS NULL
  )
);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_status_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_status_check CHECK (
  status IN ('PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'BILL_SUBMITTED', 'PROCESSED')
);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_travel_mode_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_travel_mode_check CHECK (
  travel_mode IN ('BUS', 'TRAIN', 'CAR')
);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_counts_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_counts_check CHECK (
  persons_travelling > 0 AND days_planned > 0
);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_expenses_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_expenses_check CHECK (
  food_expense >= 0
  AND local_expense >= 0
  AND accommodation_expense >= 0
  AND travel_expense >= 0
  AND requested_amount >= 0
);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_approved_amount_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_approved_amount_check CHECK (
  approved_amount IS NULL
  OR (approved_amount > 0 AND approved_amount <= requested_amount)
);

CREATE TABLE IF NOT EXISTS tour_advance_events (
  id UUID PRIMARY KEY,
  tour_advance_request_id UUID NOT NULL REFERENCES tour_advance_requests(id) ON DELETE CASCADE,
  action VARCHAR(32) NOT NULL,
  actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tour_advance_events_request
  ON tour_advance_events (tour_advance_request_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS tour_advance_events;
DROP TABLE IF EXISTS tour_advance_requests;
DROP SEQUENCE IF EXISTS tour_advance_request_number_seq;
