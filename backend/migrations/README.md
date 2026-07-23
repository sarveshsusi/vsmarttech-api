# SQL migrations (legacy reference)

These `.sql` files under `backend/migrations/` are **historical/manual** helpers.

**Source of truth for production schema:** `database/migrations/` via goose
(`MIGRATE_MODE=goose`). See `database/migrations/README.md`.

## Rules

1. Do not add new production DDL here — add a goose migration under `database/migrations/`.
2. Local development may still use `MIGRATE_MODE=auto` (GORM AutoMigrate).
3. Workers must **not** run migrations (API only).
