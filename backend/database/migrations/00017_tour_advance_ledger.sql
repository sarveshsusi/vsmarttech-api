-- +goose Up
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS credit_applied NUMERIC(12,2) NOT NULL DEFAULT 0;
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS net_payable NUMERIC(12,2);
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS approval_justification TEXT NOT NULL DEFAULT '';

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_credit_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_credit_check CHECK (
  credit_applied >= 0
  AND (approved_amount IS NULL OR credit_applied <= approved_amount)
  AND (net_payable IS NULL OR (net_payable >= 0 AND approved_amount IS NOT NULL AND net_payable <= approved_amount))
);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_payment_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_payment_check CHECK (
  (payment_mode IS NULL AND transaction_id IS NULL)
  OR (payment_mode = 'CASH' AND transaction_id IS NULL)
  OR (payment_mode = 'CARRY_FORWARD' AND transaction_id IS NULL)
  OR (
    payment_mode = 'ONLINE'
    AND transaction_id IS NOT NULL
    AND length(btrim(transaction_id)) > 0
  )
);

UPDATE tour_advance_requests
SET net_payable = approved_amount
WHERE payment_mode IS NOT NULL
  AND net_payable IS NULL
  AND approved_amount IS NOT NULL;

CREATE TABLE IF NOT EXISTS tour_advance_expenses (
  id UUID PRIMARY KEY,
  tour_advance_request_id UUID NOT NULL REFERENCES tour_advance_requests(id) ON DELETE RESTRICT,
  engineer_id UUID NOT NULL REFERENCES support_engineers(id) ON DELETE RESTRICT,
  category VARCHAR(20) NOT NULL,
  bill_type VARCHAR(10) NOT NULL,
  expense_date VARCHAR(10) NOT NULL,
  description VARCHAR(500) NOT NULL,
  claimed_amount NUMERIC(12,2) NOT NULL,
  reference_number VARCHAR(100) NOT NULL DEFAULT '',
  missing_reason VARCHAR(500) NOT NULL DEFAULT '',
  accepted_amount NUMERIC(12,2),
  status VARCHAR(32) NOT NULL,
  review_reason TEXT NOT NULL DEFAULT '',
  reviewed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tour_advance_expenses_request ON tour_advance_expenses (tour_advance_request_id);
CREATE INDEX IF NOT EXISTS idx_tour_advance_expenses_engineer ON tour_advance_expenses (engineer_id);
CREATE INDEX IF NOT EXISTS idx_tour_advance_expenses_status ON tour_advance_expenses (status);

ALTER TABLE tour_advance_expenses DROP CONSTRAINT IF EXISTS tour_advance_expense_category_check;
ALTER TABLE tour_advance_expenses ADD CONSTRAINT tour_advance_expense_category_check CHECK (
  category IN ('FOOD', 'LOCAL_TRAVEL', 'ACCOMMODATION', 'TRAVEL', 'OTHER')
);
ALTER TABLE tour_advance_expenses DROP CONSTRAINT IF EXISTS tour_advance_expense_bill_check;
ALTER TABLE tour_advance_expenses ADD CONSTRAINT tour_advance_expense_bill_check CHECK (
  (bill_type = 'BILL' AND length(btrim(reference_number)) > 0 AND missing_reason = '')
  OR (bill_type = 'NON_BILL' AND length(btrim(missing_reason)) > 0 AND reference_number = '')
);
ALTER TABLE tour_advance_expenses DROP CONSTRAINT IF EXISTS tour_advance_expense_status_check;
ALTER TABLE tour_advance_expenses ADD CONSTRAINT tour_advance_expense_status_check CHECK (
  status IN ('PENDING_REVIEW', 'NEEDS_CLARIFICATION', 'VERIFIED', 'REJECTED')
);
ALTER TABLE tour_advance_expenses DROP CONSTRAINT IF EXISTS tour_advance_expense_amount_check;
ALTER TABLE tour_advance_expenses ADD CONSTRAINT tour_advance_expense_amount_check CHECK (
  claimed_amount > 0
  AND (
    accepted_amount IS NULL
    OR (status = 'VERIFIED' AND accepted_amount >= 0 AND accepted_amount <= claimed_amount)
  )
);

CREATE TABLE IF NOT EXISTS tour_advance_disbursements (
  id UUID PRIMARY KEY,
  tour_advance_request_id UUID NOT NULL REFERENCES tour_advance_requests(id) ON DELETE RESTRICT,
  source VARCHAR(16) NOT NULL,
  amount NUMERIC(12,2) NOT NULL,
  payment_mode VARCHAR(16) NOT NULL,
  transaction_id VARCHAR(64),
  actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  idempotency_key VARCHAR(80) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_disburse_source
  ON tour_advance_disbursements (tour_advance_request_id, source);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_disburse_idempotency
  ON tour_advance_disbursements (idempotency_key);

ALTER TABLE tour_advance_disbursements DROP CONSTRAINT IF EXISTS tour_advance_disburse_check;
ALTER TABLE tour_advance_disbursements ADD CONSTRAINT tour_advance_disburse_check CHECK (
  source IN ('ORIGINAL', 'ADDITIONAL')
  AND (
    (payment_mode = 'CASH' AND transaction_id IS NULL AND amount > 0)
    OR (payment_mode = 'ONLINE' AND transaction_id IS NOT NULL AND length(btrim(transaction_id)) > 0 AND amount > 0)
    OR (payment_mode = 'CARRY_FORWARD' AND transaction_id IS NULL AND amount = 0)
  )
);

INSERT INTO tour_advance_disbursements (
  id, tour_advance_request_id, source, amount, payment_mode, transaction_id, actor_user_id, idempotency_key, created_at
)
SELECT gen_random_uuid(), id, 'ORIGINAL', approved_amount, payment_mode, transaction_id, approved_by,
       'legacy-original-' || id::text, COALESCE(approved_at, NOW())
FROM tour_advance_requests
WHERE payment_mode IS NOT NULL
  AND approved_amount IS NOT NULL
  AND approved_by IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM tour_advance_disbursements d
    WHERE d.tour_advance_request_id = tour_advance_requests.id AND d.source = 'ORIGINAL'
  );

INSERT INTO tour_advance_disbursements (
  id, tour_advance_request_id, source, amount, payment_mode, transaction_id, actor_user_id, idempotency_key, created_at
)
SELECT gen_random_uuid(), a.tour_advance_request_id, 'ADDITIONAL', a.approved_amount, a.payment_mode, a.transaction_id, a.approved_by,
       'legacy-additional-' || a.id::text, COALESCE(a.approved_at, NOW())
FROM tour_advance_additionals a
WHERE a.payment_mode IS NOT NULL
  AND a.approved_amount IS NOT NULL
  AND a.approved_by IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM tour_advance_disbursements d
    WHERE d.tour_advance_request_id = a.tour_advance_request_id AND d.source = 'ADDITIONAL'
  );

CREATE TABLE IF NOT EXISTS tour_advance_ledger (
  id UUID PRIMARY KEY,
  engineer_id UUID NOT NULL REFERENCES support_engineers(id) ON DELETE RESTRICT,
  originating_tour_id UUID NOT NULL REFERENCES tour_advance_requests(id) ON DELETE RESTRICT,
  consuming_tour_id UUID REFERENCES tour_advance_requests(id) ON DELETE RESTRICT,
  entry_type VARCHAR(32) NOT NULL,
  amount NUMERIC(12,2) NOT NULL,
  balance_before NUMERIC(12,2) NOT NULL,
  balance_after NUMERIC(12,2) NOT NULL,
  actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  reason TEXT NOT NULL DEFAULT '',
  idempotency_key VARCHAR(120) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tour_ledger_engineer ON tour_advance_ledger (engineer_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_ledger_idempotency ON tour_advance_ledger (idempotency_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_ledger_applied_origin
  ON tour_advance_ledger (consuming_tour_id, originating_tour_id)
  WHERE entry_type = 'CREDIT_APPLIED';
CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_ledger_issued_once
  ON tour_advance_ledger (originating_tour_id)
  WHERE entry_type = 'CREDIT_ISSUED';

ALTER TABLE tour_advance_ledger DROP CONSTRAINT IF EXISTS tour_advance_ledger_check;
ALTER TABLE tour_advance_ledger ADD CONSTRAINT tour_advance_ledger_check CHECK (
  amount > 0
  AND balance_before >= 0
  AND balance_after >= 0
  AND (
    (entry_type = 'CREDIT_ISSUED' AND consuming_tour_id IS NULL AND balance_after = balance_before + amount)
    OR (entry_type = 'CREDIT_APPLIED' AND consuming_tour_id IS NOT NULL AND balance_after = balance_before - amount)
  )
);

CREATE TABLE IF NOT EXISTS tour_advance_flags (
  id UUID PRIMARY KEY,
  tour_advance_request_id UUID NOT NULL REFERENCES tour_advance_requests(id) ON DELETE RESTRICT,
  kind VARCHAR(32) NOT NULL,
  reason VARCHAR(300) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tour_advance_flags_request ON tour_advance_flags (tour_advance_request_id);

CREATE TABLE IF NOT EXISTS tour_advance_policies (
  id UUID PRIMARY KEY,
  high_value_threshold NUMERIC(12,2),
  updated_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  singleton BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_advance_policies_singleton ON tour_advance_policies (singleton);

ALTER TABLE tour_advance_policies DROP CONSTRAINT IF EXISTS tour_advance_policy_threshold_check;
ALTER TABLE tour_advance_policies ADD CONSTRAINT tour_advance_policy_threshold_check CHECK (
  high_value_threshold IS NULL OR high_value_threshold > 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tour_advance_one_open
  ON tour_advance_requests (engineer_id)
  WHERE status NOT IN ('REJECTED', 'PROCESSED');

-- +goose Down
DROP INDEX IF EXISTS idx_tour_advance_one_open;
DROP TABLE IF EXISTS tour_advance_policies;
DROP TABLE IF EXISTS tour_advance_flags;
DROP TABLE IF EXISTS tour_advance_ledger;
DROP TABLE IF EXISTS tour_advance_disbursements;
DROP TABLE IF EXISTS tour_advance_expenses;
ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_credit_check;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS approval_justification;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS net_payable;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS credit_applied;
