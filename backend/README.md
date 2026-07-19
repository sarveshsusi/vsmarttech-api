# VSmart Tech API (backend)

Go/Gin CRM API: tickets, AMC, contracts, RBAC, 2FA. Modular monolith with optional worker containers.

## Quick start

```bash
cp .env.example .env
# edit .env — set JWT secrets, DATABASE_URL, AWS if using S3

go test ./...
go run ./cmd/api
```

Compose (API + workers + Postgres + Nginx on `:8081`):

```bash
docker compose up --build
curl -s http://localhost:8081/healthz
```

## Binaries

| Path | Role |
|---|---|
| `cmd/api` | HTTP API |
| `cmd/worker-sla` | Hourly SLA escalation |
| `cmd/worker-contracts` | Daily contract expiry notifications |
| `main` | Legacy alias of `cmd/api` |

## Domain modules

See [ARCHITECTURE.md](./ARCHITECTURE.md) and `internal/modules/`.

## Security notes

- Never commit `.env` or `.env.production` with real secrets.
- Production requires non-default `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET`.
- Nginx is the only public entrypoint; Postgres stays on the internal network.
