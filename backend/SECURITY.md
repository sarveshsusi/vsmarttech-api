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

- Access tokens: Bearer JWT (HS256), short TTL.
- Every authenticated request re-loads `is_active` + role from the database (disabled accounts cannot keep using a JWT).
- Refresh tokens: HttpOnly cookie, `SameSite=Strict`, `Secure` in production; rotated on refresh.
- Roles: `admin` | `support` | `customer`.
- Login: rate-limited (10/min) + IP brute-force guard (5 fails → 15m lock).
- OTP verify / forgot-password / reset-password: rate-limited (5/min).
- Refresh: rate-limited (30/min).
- Password policy: 8+ chars with upper, lower, number, and special character.

## Authorization / IDOR

- Customer ticket fetch resolves `customers` by JWT `user_id`, then compares `ticket.customer_id` to `customer.id` (never compare to `users.id`).
- Feedback submit binds to the caller, verifies ticket ownership + closed status, and uses the ticket’s assigned engineer (client cannot spoof `engineer_id`).
- Notification mark-read requires `id` **and** `user_id`.

## Uploads

- Proof uploads: 1MB max, magic-byte sniff (JPEG/PNG/GIF/WebP), image decode required, blocked dangerous extensions, random server-side filenames.
- Do not trust client `Content-Type`.

## Reporting

If you suspect a leaked secret in git history, rotate credentials and scrub history (BFG / `git filter-repo`) before any public fork.
