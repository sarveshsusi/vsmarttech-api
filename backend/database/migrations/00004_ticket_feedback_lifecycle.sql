-- +goose Up
ALTER TABLE ticket_feedbacks ADD COLUMN IF NOT EXISTS customer_id UUID;
ALTER TABLE ticket_feedbacks ADD COLUMN IF NOT EXISTS company_id UUID;
ALTER TABLE ticket_feedbacks ADD COLUMN IF NOT EXISTS remarks VARCHAR(500) NOT NULL DEFAULT '';
ALTER TABLE ticket_feedbacks ADD COLUMN IF NOT EXISTS feedback_status VARCHAR(20) NOT NULL DEFAULT 'Pending';
ALTER TABLE ticket_feedbacks ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ;
ALTER TABLE ticket_feedbacks ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE ticket_feedbacks ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Migrate legacy comment → remarks
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'ticket_feedbacks' AND column_name = 'comment'
  ) THEN
    EXECUTE $sql$
      UPDATE ticket_feedbacks
      SET remarks = LEFT(COALESCE(NULLIF(TRIM(comment), ''), remarks), 500)
      WHERE (remarks IS NULL OR remarks = '') AND comment IS NOT NULL AND TRIM(comment) <> ''
    $sql$;
  END IF;
END $$;

-- Existing rows with a rating are Submitted
UPDATE ticket_feedbacks
SET
  feedback_status = 'Submitted',
  submitted_at = COALESCE(submitted_at, created_at),
  updated_at = COALESCE(updated_at, created_at, NOW())
WHERE rating IS NOT NULL AND rating >= 1;

-- Allow null rating for Pending rows (legacy NOT NULL may exist)
ALTER TABLE ticket_feedbacks ALTER COLUMN rating DROP NOT NULL;

-- Backfill customer_id / company_id from tickets + customers
UPDATE ticket_feedbacks tf
SET
  customer_id = t.customer_id,
  company_id = c.company_id
FROM tickets t
JOIN customers c ON c.id = t.customer_id
WHERE tf.ticket_id = t.id
  AND (tf.customer_id IS NULL OR tf.company_id IS NULL);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_feedbacks_ticket_id_unique
  ON ticket_feedbacks (ticket_id);

CREATE INDEX IF NOT EXISTS idx_ticket_feedbacks_engineer_id ON ticket_feedbacks (engineer_id);
CREATE INDEX IF NOT EXISTS idx_ticket_feedbacks_customer_id ON ticket_feedbacks (customer_id);
CREATE INDEX IF NOT EXISTS idx_ticket_feedbacks_company_id ON ticket_feedbacks (company_id);
CREATE INDEX IF NOT EXISTS idx_ticket_feedbacks_status ON ticket_feedbacks (feedback_status);

-- +goose Down
DROP INDEX IF EXISTS idx_ticket_feedbacks_status;
DROP INDEX IF EXISTS idx_ticket_feedbacks_company_id;
DROP INDEX IF EXISTS idx_ticket_feedbacks_customer_id;
DROP INDEX IF EXISTS idx_ticket_feedbacks_engineer_id;
DROP INDEX IF EXISTS idx_ticket_feedbacks_ticket_id_unique;

ALTER TABLE ticket_feedbacks DROP COLUMN IF EXISTS metadata;
ALTER TABLE ticket_feedbacks DROP COLUMN IF EXISTS updated_at;
ALTER TABLE ticket_feedbacks DROP COLUMN IF EXISTS submitted_at;
ALTER TABLE ticket_feedbacks DROP COLUMN IF EXISTS feedback_status;
ALTER TABLE ticket_feedbacks DROP COLUMN IF EXISTS remarks;
ALTER TABLE ticket_feedbacks DROP COLUMN IF EXISTS company_id;
ALTER TABLE ticket_feedbacks DROP COLUMN IF EXISTS customer_id;
