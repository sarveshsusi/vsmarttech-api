-- Migration: Fix customer_solutions with incorrect customer_id references
-- This migration fixes a data issue where customer_solutions were linked to user_ids instead of customer_ids
-- This was causing foreign key constraint violations when creating tickets

-- Fix: Update customer_solutions that have user_id instead of customer_id
-- The issue: customer_solutions.customer_id should reference customers.id, not users.id
-- Solution: Find the correct customer_id for each user_id and update the reference

UPDATE customer_solutions cs
SET customer_id = c.id
FROM customers c
WHERE cs.customer_id = c.user_id
  AND c.id IN (
    -- Find customers where their user_id exists as a customer_id in customer_solutions
    SELECT DISTINCT c2.id 
    FROM customers c2 
    WHERE c2.user_id IN (
      SELECT DISTINCT customer_id FROM customer_solutions 
      WHERE customer_id IN (SELECT id FROM users)
    )
  );

-- Verify the fix
-- All customer_id values should now be actual customer IDs from the customers table
SELECT COUNT(*) as invalid_references
FROM customer_solutions cs
WHERE NOT EXISTS (
  SELECT 1 FROM customers c WHERE c.id = cs.customer_id
);
-- This should return 0 if the fix is successful
