# Deployment Guide

## Environment
Set:
- `PORT`
- `GOOGLE_CLIENT_ID`
- `JWT_SECRET`
- `DATABASE_URL`
- `FRONTEND_ORIGIN` (explicitly set allowed origins)
- `DB_MAX_CONNS`
- `ADMIN_EMAILS`

## Database
1. Provision PostgreSQL.
2. Apply migrations in `database/migrations/` in lexical order.
3. Ensure indexes from `001_init_schema.sql` exist for leaderboard and adaptive-question queries.

## Runtime hardening
- Keep `JWT_SECRET` out of source control.
- Use HTTPS behind a reverse proxy.
- Restrict `FRONTEND_ORIGIN` to known domains only.
- Keep `/auth/login` and `/auth/register` behind per-IP rate limits (already enabled in app).

## Health checks
- Liveness/readiness endpoint: `GET /health`.
- Monitor DB pool saturation and request latency.
