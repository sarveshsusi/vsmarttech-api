# VSmart API architecture

## Current shape: modular monolith + workers

```
Client → Nginx:8081 (gateway) → api:8080 → Postgres (single DB)
                              ↘ worker-sla
                              ↘ worker-contracts
```

- **One public URL** (`/api/v1/...`). The frontend must not talk to multiple origins.
- **One Postgres database** shared by API and workers.
- Domain packages under `internal/modules/{auth,crm,tickets,amc,notify}` own route registration.
- Process entrypoints: `cmd/api`, `cmd/worker-sla`, `cmd/worker-contracts`.

## Explicitly deferred (do not do yet)

| Decision | Status | Why |
|---|---|---|
| DB-per-service | **Deferred** | Tickets/AMC/CRM join users & companies; splitting DBs needs events/saga work with little payoff on one host |
| Kubernetes | **Deferred** | Compose/NAS is enough until multi-host scale or multi-team ownership appears |
| Full HTTP microservice split | **Deferred** | Extract workers first; keep API as one binary until a concrete scale/team driver |

## When to reconsider a service extract

Only after:

1. CI + auth/ticket tests are green
2. Secrets are not in git
3. Workers have been stable in Compose
4. There is a real reason (independent scale, separate team ownership, or deploy isolation that workers do not solve)

First API extract candidates (still **one Postgres**): `auth-identity`, then `tickets`.

## Local vs Compose crons

| Mode | `RUN_INPROCESS_CRONS` | Who runs SLA / contract jobs |
|---|---|---|
| `go run ./cmd/api` (dev default) | `true` (default) | Inside API process |
| Docker Compose (this repo) | `false` | `worker-sla` + `worker-contracts` |

Never run both in-process crons and worker containers against the same DB.
