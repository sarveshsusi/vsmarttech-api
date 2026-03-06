-- Add support_comment column to tickets table
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS support_comment TEXT;

-- Add index for filtering by support_comment (optional, for future queries)
CREATE INDEX IF NOT EXISTS idx_tickets_support_comment ON tickets(support_comment);
