-- +goose Up
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS payment_mode VARCHAR(10);
ALTER TABLE tour_advance_requests
  ADD COLUMN IF NOT EXISTS transaction_id VARCHAR(64);

ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_payment_check;
ALTER TABLE tour_advance_requests ADD CONSTRAINT tour_advance_payment_check CHECK (
  (payment_mode IS NULL AND transaction_id IS NULL)
  OR (payment_mode = 'CASH' AND transaction_id IS NULL)
  OR (
    payment_mode = 'ONLINE'
    AND transaction_id IS NOT NULL
    AND length(btrim(transaction_id)) > 0
  )
);

ALTER TABLE tour_advance_additionals
  ADD COLUMN IF NOT EXISTS payment_mode VARCHAR(10);
ALTER TABLE tour_advance_additionals
  ADD COLUMN IF NOT EXISTS transaction_id VARCHAR(64);

ALTER TABLE tour_advance_additionals DROP CONSTRAINT IF EXISTS tour_advance_additional_payment_check;
ALTER TABLE tour_advance_additionals ADD CONSTRAINT tour_advance_additional_payment_check CHECK (
  (payment_mode IS NULL AND transaction_id IS NULL)
  OR (payment_mode = 'CASH' AND transaction_id IS NULL)
  OR (
    payment_mode = 'ONLINE'
    AND transaction_id IS NOT NULL
    AND length(btrim(transaction_id)) > 0
  )
);

-- +goose Down
ALTER TABLE tour_advance_additionals DROP CONSTRAINT IF EXISTS tour_advance_additional_payment_check;
ALTER TABLE tour_advance_additionals DROP COLUMN IF EXISTS transaction_id;
ALTER TABLE tour_advance_additionals DROP COLUMN IF EXISTS payment_mode;
ALTER TABLE tour_advance_requests DROP CONSTRAINT IF EXISTS tour_advance_payment_check;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS transaction_id;
ALTER TABLE tour_advance_requests DROP COLUMN IF EXISTS payment_mode;
