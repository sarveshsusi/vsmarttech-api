-- Add chargeable_type field to customer_solutions table
-- This field is used for "Others/Chargeable" contract types

ALTER TABLE customer_solutions
ADD COLUMN chargeable_type VARCHAR(20);

-- Add comment for documentation
COMMENT ON COLUMN customer_solutions.chargeable_type IS 'Type of chargeable service: Chargeable or Others';
