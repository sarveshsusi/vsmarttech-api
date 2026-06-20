-- Migration to change ticket ID from UUID to VARCHAR format (VS/MM/YY/number)
-- Run this migration AFTER backing up your database

-- Step 1: Create new tables with VARCHAR ticket_id

-- Create temporary tickets table with new ID format
CREATE TABLE IF NOT EXISTS tickets_new (
    id VARCHAR(20) PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id),
    customer_solution_id UUID REFERENCES customer_solutions(id),
    engineer_id UUID REFERENCES support_engineers(id),
    
    solution_title VARCHAR(150),
    po_number VARCHAR(100),
    contract_type VARCHAR(20),
    
    title VARCHAR(255) NOT NULL,
    description TEXT,
    
    status VARCHAR(50) CHECK (status IN ('Open', 'Assigned', 'In Progress', 'Closed')) DEFAULT 'Open',
    
    priority VARCHAR(30),
    support_mode VARCHAR(50),
    service_call_type VARCHAR(50),
    
    closure_proof_image TEXT,
    support_comment TEXT,
    
    sla_hours INT,
    target_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Note: This migration requires manual data migration
-- After running this, you will need to:
-- 1. Migrate existing ticket data to new format (optional - or start fresh)
-- 2. Drop old tickets table and rename tickets_new to tickets
-- 3. Update related tables (ticket_assignments, ticket_status_histories, etc.)

-- For a fresh start (recommended for development):
-- DROP TABLE IF EXISTS tickets CASCADE;
-- ALTER TABLE tickets_new RENAME TO tickets;

-- Create indexes on new table
CREATE INDEX IF NOT EXISTS idx_tickets_customer_id ON tickets_new(customer_id);
CREATE INDEX IF NOT EXISTS idx_tickets_engineer_id ON tickets_new(engineer_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets_new(status);
CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets_new(created_at DESC);

-- Update related tables to use VARCHAR for ticket_id
-- (Run these AFTER migrating tickets table)

-- ALTER TABLE ticket_assignments ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE ticket_status_histories ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE ticket_comments ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE ticket_attachments ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE ticket_feedbacks ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE notifications ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE webhook_events ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE ticket_escalations ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE service_visits ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE digital_signatures ALTER COLUMN ticket_id TYPE VARCHAR(20);
-- ALTER TABLE amc_schedules ALTER COLUMN ticket_id TYPE VARCHAR(20);
