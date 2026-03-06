-- Fix existing customers with empty names
-- Set customer name to contact_person if name is empty or null
UPDATE customers
SET name = contact_person
WHERE name IS NULL OR name = '';
