# Versioned migrations (goose)

Production schema changes are applied with [goose](https://github.com/pressly/goose).
SQL files live in `database/migrations/` and are embedded into the API binary.

## Modes (`MIGRATE_MODE`)

| Value | When | Behavior |
|-------|------|----------|
| `goose` | **production default** | Run goose `Up`. If DB has no `users` table, AutoMigrate once for baseline, then goose. |
| `auto` | **development default** | GORM AutoMigrate only (convenient for local iteration). |

Workers never migrate (`Migrate: false`).

## Rules

1. New production DDL goes in a numbered goose migration under `database/migrations/`.
2. Prefer additive changes (`ADD COLUMN IF NOT EXISTS`, backfills). Avoid drops/renames in the same release as app code that still needs the column.
3. Never leave empty migration files.
4. Legacy SQL under `migrations/` (repo root of backend) is historical reference only — do not run against production.
5. After changing models locally with `MIGRATE_MODE=auto`, add an equivalent goose migration before deploying to production.

## Current forward migrations

- `00001_baseline.sql` — records historical AutoMigrate schema
- `00002_refresh_token_family_id.sql` — refresh token session families
