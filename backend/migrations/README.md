# SQL migrations (reference)

These `.sql` files are **manual/reference** helpers. The running API applies schema via GORM AutoMigrate in `database.Migrate` on `cmd/api` boot.

## Rules

1. Prefer model changes + AutoMigrate for additive columns.
2. Use SQL files for data backfills or constraints AutoMigrate cannot express.
3. Never leave empty migration files.
4. Workers must **not** run AutoMigrate (API only).
5. Document any production SQL in the PR that introduces it.
