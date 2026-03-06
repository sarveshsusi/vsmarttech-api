-- Migration: Make customer_solution_id optional in tickets table
-- This allows customers to create tickets without selecting a customer_solution

ALTER TABLE tickets 
ALTER COLUMN customer_solution_id DROP NOT NULL;
