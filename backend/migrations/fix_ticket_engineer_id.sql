-- Update tickets that have assignments but no engineer_id set
UPDATE tickets t
SET engineer_id = ta.engineer_id
FROM ticket_assignments ta
WHERE t.id = ta.ticket_id
AND t.engineer_id IS NULL;
