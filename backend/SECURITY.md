# Security

## Secrets

- Never commit `.env`, `.env.production`, or real credentials.
- Use [`.env.example`](./.env.example) and [`.env.production.example`](./.env.production.example) as templates.
- Production (`APP_ENV=production`) **requires** non-default `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET`.
- If `.env.production` or binaries were ever committed historically, **rotate** JWT, DB, mail, and AWS keys immediately.

## HTTP surface

- Public entry: Nginx only (Compose port `8081`).
- Postgres is internal-network only.
- CORS + security headers are applied **once** in `internal/bootstrap` (not duplicated in routes).
- Trusted proxies default to Docker/Nginx ranges via `TRUSTED_PROXIES`.

## Auth

- Access tokens: Bearer JWT (HS256).
- Refresh tokens: HttpOnly cookie (Secure in production).
- Roles: `admin` | `support` | `customer`.
- Login endpoints are rate-limited and brute-force guarded (in-memory; use Redis before multi-replica).

## Reporting

If you suspect a leaked secret in git history, rotate credentials and scrub history (BFG / `git filter-repo`) before any public fork.
