-- +goose Up
CREATE TABLE IF NOT EXISTS amc_customer_name_histories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_solution_id UUID NOT NULL,
  customer_id UUID NOT NULL,
  po_number VARCHAR(100) NOT NULL,
  old_name VARCHAR(150) NOT NULL,
  new_name VARCHAR(150) NOT NULL,
  changed_by UUID NOT NULL,
  changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  note VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_amc_name_hist_solution ON amc_customer_name_histories (customer_solution_id);
CREATE INDEX IF NOT EXISTS idx_amc_name_hist_customer ON amc_customer_name_histories (customer_id);
CREATE INDEX IF NOT EXISTS idx_amc_name_hist_po ON amc_customer_name_histories (po_number);
CREATE INDEX IF NOT EXISTS idx_amc_name_hist_changed_at ON amc_customer_name_histories (changed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_amc_name_hist_changed_at;
DROP INDEX IF EXISTS idx_amc_name_hist_po;
DROP INDEX IF EXISTS idx_amc_name_hist_customer;
DROP INDEX IF EXISTS idx_amc_name_hist_solution;
DROP TABLE IF EXISTS amc_customer_name_histories;
