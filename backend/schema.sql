-- =========================================================
-- 1. SETUP & EXTENSIONS
-- =========================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =========================================================
-- 2. USER MANAGEMENT & AUTH
-- =========================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100),
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'support', 'customer')),
    is_active BOOLEAN DEFAULT TRUE,
    must_reset_password BOOLEAN DEFAULT FALSE,
    created_by UUID,

    two_fa_enabled BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    last_otp_verified_at TIMESTAMPTZ,
    last_password_reset_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_created_by ON users(created_by);

-- =========================================================
-- TOKENS & SECURITY
-- =========================================================

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS two_fa_otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    code TEXT,
    expires_at TIMESTAMPTZ,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS remembered_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =========================================================
-- 3. PROFILES
-- =========================================================

CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    company VARCHAR(150),
    phone VARCHAR(20),
    address TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS support_engineers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    designation VARCHAR(100),
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =========================================================
-- 4. SOLUTIONS
-- =========================================================

CREATE TABLE IF NOT EXISTS solutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(150) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =========================================================
-- 5. CUSTOMER SOLUTIONS (PO + CONTRACT)
-- =========================================================

CREATE TABLE IF NOT EXISTS customer_solutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    solution_id UUID NOT NULL REFERENCES solutions(id),

    po_number VARCHAR(100) NOT NULL UNIQUE,
    contract_type VARCHAR(20) NOT NULL CHECK (contract_type IN ('AMC', 'Warranty')),

    -- AMC
    amc_type VARCHAR(50),
    amc_start_date TIMESTAMPTZ,
    amc_end_date TIMESTAMPTZ,

    -- Warranty
    warranty_start_date TIMESTAMPTZ,
    warranty_end_date TIMESTAMPTZ,

    is_active BOOLEAN DEFAULT TRUE,
    assigned_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customer_solutions_customer
ON customer_solutions(customer_id);

-- =========================================================
-- 6. TICKETING SYSTEM (PO BASED)
-- =========================================================

CREATE TABLE IF NOT EXISTS tickets (
    id VARCHAR(20) PRIMARY KEY,

    customer_id UUID NOT NULL REFERENCES customers(id),
    customer_solution_id UUID REFERENCES customer_solutions(id),

    -- 🔒 SNAPSHOT (IMMUTABLE)
    solution_title VARCHAR(150),
    po_number VARCHAR(100),
    contract_type VARCHAR(20),

    title VARCHAR(255) NOT NULL,
    description TEXT,

    status VARCHAR(50)
        CHECK (status IN ('Open', 'Assigned', 'In Progress', 'Closed'))
        DEFAULT 'Open',

    priority VARCHAR(30),
    support_mode VARCHAR(50),
    service_call_type VARCHAR(50),

    closure_proof_image TEXT,

    sla_hours INT,
    target_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =========================================================
-- 7. SEED SUPER ADMIN (SAFE)
-- =========================================================

INSERT INTO users (
    id,
    name,
    email,
    password,
    role,
    is_active,
    must_reset_password,
    two_fa_enabled,
    created_by,
    created_at,
    updated_at
)
VALUES (
    gen_random_uuid(),
    'Super Admin',
    'admin@yourapp.com',
    '$2a$10$fjBmX7zDk/MEYjI9ajz8euB/uOqabx9ic.KupNC6b0c1h3tulCCsy',
    'admin',
    TRUE,
    FALSE,
    FALSE,
    NULL,
    NOW(),
    NOW()
)
ON CONFLICT (email) DO NOTHING;

UPDATE users
SET two_fa_enabled = FALSE
WHERE email = 'admin@yourapp.com';
