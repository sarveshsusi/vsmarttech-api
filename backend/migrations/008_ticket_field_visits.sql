-- Reference: ticket field visits (applied via GORM AutoMigrate).
-- service_visits is extended with visit_date + relations.
-- New: service_visit_proofs, service_visit_co_engineers (many2many).

-- Optional production backfill if start_time was NOT NULL and visit_date is added empty:
-- UPDATE service_visits SET visit_date = COALESCE(visit_date, start_time::date) WHERE visit_date IS NULL;
