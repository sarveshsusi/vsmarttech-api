-- Migration: Add unique constraint to customer_products table
-- Run this in pgAdmin or via psql

-- First, remove any duplicate rows if they exist
DELETE FROM customer_products a USING customer_products b
WHERE a.id > b.id 
  AND a.customer_id = b.customer_id 
  AND a.product_id = b.product_id;

-- Then add the unique constraint
ALTER TABLE customer_products 
ADD CONSTRAINT customer_products_customer_id_product_id_key 
UNIQUE (customer_id, product_id);
