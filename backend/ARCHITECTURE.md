# VSmart API architecture

## Current shape: modular monolith + workers

```
Client (Vercel SPA) → Nginx:8081 (gateway) → api:8080 → Postgres (single DB)
                                            ↘ worker-sla
                                            ↘ worker-contracts
```

- **One public URL** (`/api/v1/...`). Frontend must not use multiple API origins.
- **One Postgres** shared by API and workers (DB-per-service deferred).
- Domain packages: `internal/modules/{auth,crm,tickets,amc,notify}`.
- Wiring: `internal/bootstrap`. Entrypoints: `cmd/api`, `cmd/worker-sla`, `cmd/worker-contracts`.
- Observability: `X-Request-ID`, structured `slog` access logs, audit rows in `audit_logs`, `/healthz` + `/readyz`.

## Explicitly deferred

| Decision | Status |
|---|---|
| DB-per-service | Deferred |
| Kubernetes | Deferred |
| Full HTTP microservice split | Deferred |

## Migrations

| Mechanism | Role |
|---|---|
| `database.Migrate` (GORM AutoMigrate) | Source of truth at API boot |
| `migrations/*.sql` | Manual/reference only; keep in sync with models; no empty files |
| `schema.sql` | Snapshot reference |

Before changing schema in production: migrate staging first, confirm AutoMigrate is safe for the change, then deploy API (workers do not migrate).

## Local vs Compose crons

| Mode | `RUN_INPROCESS_CRONS` | Jobs |
|---|---|---|
| `go run ./cmd/api` | `true` (default) | Inside API |
| Docker Compose | `false` | `worker-sla` + `worker-contracts` |

Never run both against the same DB.

## Deploy notes (AWS later)

Keep Compose topology: ALB/Nginx → API; workers as separate tasks/containers; RDS for Postgres; S3 for uploads; set `FRONTEND_URL` to the Vercel URL.
