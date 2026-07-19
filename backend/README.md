# VSmart Tech API (backend)

Go/Gin CRM API: tickets, AMC, contracts, RBAC, 2FA.

**Architecture:** modular monolith (one HTTP API) + optional worker containers. **Not** full microservices. Frontend (e.g. Vercel) talks to a **single** public API URL.

See also: [ARCHITECTURE.md](./ARCHITECTURE.md) · [SECURITY.md](./SECURITY.md) · [openapi.yaml](./openapi.yaml)

## Quick start (local)

```bash
cp .env.example .env
# set DATABASE_URL, JWT_*, FRONTEND_URL, AWS_* if using S3

go test ./...
go run ./cmd/api
# default: RUN_INPROCESS_CRONS=true (SLA + contract jobs inside API)
```

## Docker Compose

```bash
docker compose up --build -d
docker compose ps
curl -s http://localhost:8081/healthz   # liveness via Nginx
curl -s http://localhost:8081/readyz    # readiness (DB ping)
```

Services: `api`, `worker-sla`, `worker-contracts`, `postgres`, `nginx` (`:8081`).  
Compose sets `RUN_INPROCESS_CRONS=false` so workers own cron jobs.

## Processes

| Path | Role |
|---|---|
| `cmd/api` | HTTP API + AutoMigrate |
| `cmd/worker-sla` | Hourly SLA escalation |
| `cmd/worker-contracts` | Daily contract expiry |
| `main` | Legacy alias of `cmd/api` |

## Domain modules (`internal/modules`)

| Module | Owns |
|---|---|
| auth | login, JWT, 2FA, users |
| crm | companies, customers, solutions, dashboards |
| tickets | tickets, feedback, uploads |
| amc | AMC assignments/visits, contracts |
| notify | in-app notifications |

## Route catalog (high level)

| Prefix | Auth |
|---|---|
| `POST /api/v1/auth/*` | Public |
| `GET/POST /api/v1/profile`, `/2fa/*`, `/logout` | JWT |
| `/api/v1/admin/*` | JWT + admin |
| `/api/v1/support/*` | JWT + support |
| `/api/v1/customer/*` | JWT + customer |
| `/api/v1/notifications/*` | JWT |
| `GET /healthz`, `GET /readyz` | Public |

Full OpenAPI stub: [openapi.yaml](./openapi.yaml).

## Vercel frontend + this API

- Host **frontend on Vercel**; host **this API + Postgres on AWS/NAS/Docker** (not Vercel).
- Set Vercel env `VITE_API_URL=https://api.yourdomain.com` (or your public Nginx URL).
- Set backend `FRONTEND_URL=https://your-app.vercel.app` for CORS and email links.

## Migrations

- Runtime schema: GORM `AutoMigrate` on API boot (`database.Migrate`).
- SQL under `migrations/` is reference/manual only — do not run conflicting ad-hoc SQL without coordinating AutoMigrate.
- Checklist before production schema change: add model field → AutoMigrate in staging → document in PR → avoid empty/orphan migration files.

## Security

- Never commit real `.env` / `.env.production` or binaries.
- Production requires strong JWT secrets (`APP_ENV=production`).
- Details: [SECURITY.md](./SECURITY.md).
