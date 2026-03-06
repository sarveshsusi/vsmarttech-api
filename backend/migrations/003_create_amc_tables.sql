-- Create AMC Assignments Table
CREATE TABLE amc_assignments (
    id UUID PRIMARY KEY,
    customer_solution_id UUID NOT NULL REFERENCES customer_solutions(id),
    support_engineer_id UUID NOT NULL REFERENCES support_engineers(id),
    assigned_by UUID NOT NULL,
    assigned_at TIMESTAMP NOT NULL,
    amc_start_date TIMESTAMP NOT NULL,
    amc_end_date TIMESTAMP NOT NULL,
    status VARCHAR(50) DEFAULT 'active', -- active, completed, expired
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_amc_assignments_engineer ON amc_assignments(support_engineer_id);
CREATE INDEX idx_amc_assignments_solution ON amc_assignments(customer_solution_id);
CREATE INDEX idx_amc_assignments_status ON amc_assignments(status);

-- Create AMC Visits Table
CREATE TABLE amc_visits (
    id UUID PRIMARY KEY,
    amc_assignment_id UUID NOT NULL REFERENCES amc_assignments(id) ON DELETE CASCADE,
    quarter_start_date TIMESTAMP NOT NULL,
    quarter_end_date TIMESTAMP NOT NULL,
    visit_scheduled_for TIMESTAMP NOT NULL,
    visit_date TIMESTAMP,
    status VARCHAR(50) DEFAULT 'pending', -- pending, completed, overdue
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_amc_visits_assignment ON amc_visits(amc_assignment_id);
CREATE INDEX idx_amc_visits_status ON amc_visits(status);

-- Create AMC Visit Proof Table
CREATE TABLE amc_visit_proofs (
    id UUID PRIMARY KEY,
    amc_visit_id UUID NOT NULL REFERENCES amc_visits(id) ON DELETE CASCADE,
    image_path VARCHAR(255) NOT NULL,
    description TEXT,
    uploaded_by UUID NOT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_amc_visit_proofs_visit ON amc_visit_proofs(amc_visit_id);
