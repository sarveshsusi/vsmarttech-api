# VSmart Tech API

Backend repository for the **VSmart CRM** platform: tickets, AMC site visits, contracts, companies/customers, RBAC, and 2FA.

This repo hosts the **Go API**, **background workers**, **PostgreSQL** (via Docker Compose), and **Nginx** gateway. The React frontend lives in a separate repo and is typically deployed on **Vercel**.

| Item | Value |
|---|---|
| Module path | `rbac` |
| Go version | `1.25.12` |
| Primary branch (deploy) | `master` |
| Docs / mirror branch | `main` |
| Public API (Compose) | `http://localhost:8081` via Nginx |
| Health | `GET /healthz`, `GET /readyz` |

---

## Workspace layout

```text
vsmarttech-api/
├── .github/workflows/
│   ├── ci.yml                 # vet, test, govulncheck, build
│   └── deploy.yml             # SSH deploy to Lightsail on push → master
├── .gitignore
└── backend/                   # ← application root (work from here)
    ├── cmd/
    │   ├── api/               # HTTP API + AutoMigrate
    │   ├── worker-sla/        # Hourly SLA escalation
    │   └── worker-contracts/  # Daily contract expiry
    ├── internal/
    │   ├── bootstrap/         # DI / wiring
    │   └── modules/           # Domain route ownership
    │       ├── auth/
    │       ├── crm/
    │       ├── tickets/
    │       ├── amc/
    │       └── notify/
    ├── config/                # Env loading + production guards
    ├── database/              # GORM init, pool, AutoMigrate
    ├── domain/                # Pure domain rules (e.g. ticket FSM)
    ├── handler/               # HTTP handlers
    ├── service/               # Business logic
    ├── repository/            # Data access
    ├── middleware/            # Auth, CORS, rate limit, request ID, audit
    ├── models/                # GORM models
    ├── jobs/                  # Cron workers
    ├── routes/                # Route assembly
    ├── nginx/conf/            # Reverse proxy config
    ├── migrations/            # SQL reference (manual; AutoMigrate is source of truth)
    ├── scripts/
    │   ├── backup.sh          # Daily pg_dump (+ optional S3)
    │   └── update.sh          # Pull/build with rollback
    ├── docker-compose.yml
    ├── docker-compose.prod-caddy.yml
    ├── Dockerfile
    ├── openapi.yaml
    ├── schema.sql
    ├── ARCHITECTURE.md
    ├── SECURITY.md
    ├── DEPLOY_LIGHTSAIL.md
    ├── .env.example
    └── .env.production.example
```

Frontend (separate workspace, not in this repo):

```text
vsmarttech-frontend/     # Vite + React → Vercel
  VITE_API_URL=…         # Points at this API’s public URL
```

---

## Architecture

**Modular monolith + workers** (not microservices):

```text
Browser (Vercel SPA)
        │
        │  VITE_API_URL
        ▼
   Nginx :8081  ──►  api :8080  ──►  Postgres (single DB)
                         ▲
              worker-sla ┘
         worker-contracts ┘
```

- One public origin for all `/api/v1/...` routes
- One shared Postgres (DB-per-service deferred)
- Uploads: AWS S3 in production (`STORAGE_TYPE=s3`)
- Cron jobs: Compose workers when `RUN_INPROCESS_CRONS=false`

Details: [`backend/ARCHITECTURE.md`](./backend/ARCHITECTURE.md)

---

## Domain modules

| Module | Responsibility |
|---|---|
| `auth` | Login, JWT, refresh cookies, 2FA, user admin |
| `crm` | Companies, customers, solutions, dashboards, engineers |
| `tickets` | Tickets, feedback, proof image upload |
| `amc` | AMC assignments / visits, contracts |
| `notify` | In-app notifications |

Roles: `admin` · `support` · `customer`

---

## API surface (high level)

| Prefix | Auth |
|---|---|
| `POST /api/v1/auth/*` | Public (rate-limited / brute-force guarded) |
| Profile, 2FA, logout | JWT |
| `/api/v1/admin/*` | JWT + admin |
| `/api/v1/support/*` | JWT + support |
| `/api/v1/customer/*` | JWT + customer |
| `/api/v1/notifications/*` | JWT |
| `POST /api/v1/upload/proof` | JWT |
| `GET /healthz` | Public (liveness) |
| `GET /readyz` | Public (DB ping) |

OpenAPI stub: [`backend/openapi.yaml`](./backend/openapi.yaml)

---

## Local development

### Prerequisites

- Go **1.25.12+**
- Docker + Docker Compose (optional but recommended)
- PostgreSQL (Compose provides one)

### Option A — API only (Go)

```bash
cd backend
cp .env.example .env
# Set DATABASE_URL, JWT_*, FRONTEND_URL

go test ./...
go run ./cmd/api
# RUN_INPROCESS_CRONS=true by default → SLA + contract jobs inside API
```

### Option B — Full stack (Compose)

```bash
cd backend
cp .env.example .env
docker compose up --build -d

curl -s http://localhost:8081/healthz
curl -s http://localhost:8081/readyz
```

| Service | Role | Host port |
|---|---|---|
| `nginx` | Public gateway | `8081` |
| `api` | Go HTTP API | internal `8080` |
| `worker-sla` | Hourly SLA escalation | — |
| `worker-contracts` | Daily contract expiry | — |
| `postgres` | Database (not published) | — |

Local overrides (gitignored): `backend/docker-compose.override.yml`  
Example: map API to `8090` if host `8080` is taken.

Point the frontend at the API:

```env
VITE_API_URL=http://localhost:8090
# or via Nginx:
VITE_API_URL=http://localhost:8081
```

Backend CORS / email links:

```env
FRONTEND_URL=http://localhost:5173
# also allows http://127.0.0.1:5173 in non-production
```

---

## Environment variables

Templates:

- [`backend/.env.example`](./backend/.env.example) — local
- [`backend/.env.production.example`](./backend/.env.production.example) — production

| Variable | Purpose |
|---|---|
| `APP_ENV` | `development` / `production` |
| `DATABASE_URL` | Postgres DSN (required) |
| `POSTGRES_*` | Compose Postgres bootstrap |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | Required strong secrets in production |
| `REMEMBER_DEVICE_SECRET` | Required in production |
| `FRONTEND_URL` | CORS + email deep links (real Vercel URL in prod) |
| `STORAGE_TYPE` | `local` or `s3` |
| `AWS_*` | S3 credentials / bucket (required if `s3` in prod) |
| `RUN_INPROCESS_CRONS` | `true` local API; `false` under Compose |
| `RATE_LIMIT_MAX` | Requests/minute for protected routes (default `60`) |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | Pool sizing (tight defaults for 1 GB hosts) |
| `TRUSTED_PROXIES` | Nginx / Docker CIDRs |
| `MAIL_*` | SMTP for notifications |

**Never commit** `.env` or `.env.production`. See [`backend/SECURITY.md`](./backend/SECURITY.md).

---

## Production (AWS Lightsail)

Target: Ubuntu + Docker Compose on a ~**$7 / 1 GB** Lightsail instance; frontend on Vercel.

Full step-by-step guide: [`backend/DEPLOY_LIGHTSAIL.md`](./backend/DEPLOY_LIGHTSAIL.md)

Summary:

1. Harden host (UFW, fail2ban, unattended-upgrades)
2. Install Docker from the official apt repo
3. Clone with a GitHub **deploy key** onto `master`
4. Configure `.env` from `.env.production.example`
5. `docker compose up -d --build`
6. Open firewall **22 / 80 / 443** (+ **8081** only until you have HTTPS)
7. Daily backups: `scripts/backup.sh` via cron
8. Updates: `scripts/update.sh` (or GitHub Actions deploy)
9. Optional HTTPS: Caddy + `docker-compose.prod-caddy.yml`

Compose production hardening includes:

- `restart: unless-stopped`
- Memory/CPU limits sized for 1 GB RAM
- Log rotation (`max-size` / `max-file`)
- Health checks (API, Postgres, Nginx, workers)
- Non-root API image, `read_only` + `no-new-privileges`
- Postgres tuned (`max_connections=40`, `shared_buffers=128MB`)
- **Postgres port 5432 never published**

---

## CI / CD

| Workflow | Trigger | What it does |
|---|---|---|
| [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) | Push / PR to `main`, `master`, `feature/**`, … | `go vet`, `go test`, `govulncheck`, build API + workers |
| [`.github/workflows/deploy.yml`](./.github/workflows/deploy.yml) | Push to **`master`** | SSH into Lightsail → `git reset --hard origin/master` → Compose build/up → `/healthz` |

Required GitHub secrets for deploy:

- `HOST` — Lightsail public IP / hostname  
- `USERNAME` — SSH user (usually `ubuntu`)  
- `SSH_KEY` — private key with server access  

Server checkout path expected by the workflow: `~/app/backend` on branch `master`.

---

## Testing & quality

```bash
cd backend
go vet ./...
go test ./...
```

Covered areas include config helpers, JWT/password utils, auth middleware, request IDs, ticket domain transitions, and ticket service transitions.

---

## Migrations

| Mechanism | Role |
|---|---|
| `database.Migrate` (GORM AutoMigrate) | Source of truth at **API** boot |
| `migrations/*.sql` | Manual / reference only |
| `schema.sql` | Snapshot reference |

Workers **do not** migrate. Never run in-process crons and worker containers against the same DB at once.

---

## Related docs

| Doc | Contents |
|---|---|
| [`backend/README.md`](./backend/README.md) | Backend-focused quick start |
| [`backend/ARCHITECTURE.md`](./backend/ARCHITECTURE.md) | Topology & deferred decisions |
| [`backend/SECURITY.md`](./backend/SECURITY.md) | Secrets, CORS, auth notes |
| [`backend/DEPLOY_LIGHTSAIL.md`](./backend/DEPLOY_LIGHTSAIL.md) | Production Lightsail runbook |
| [`backend/openapi.yaml`](./backend/openapi.yaml) | API stub |

---

## License / ownership

Private VSmart Tech CRM backend. Contact the repo maintainers for access and deployment credentials.
