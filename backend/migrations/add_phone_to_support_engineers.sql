-- Add phone column to support_engineers table
ALTER TABLE support_engineers
ADD COLUMN IF NOT EXISTS phone VARCHAR(20) DEFAULT '';
