-- +goose Up
-- One-time reclaim of audit_logs disk, then index for monthly retention deletes.
TRUNCATE TABLE audit_logs;
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_logs_created_at;
