# QuizArena Backend

Adaptive quiz backend with Elo-based progression, JWT/Google auth, PostgreSQL persistence, leaderboard APIs, and admin question management.

## Quick start
1. Start PostgreSQL:
   ```bash
   docker compose up -d
   ```
2. Configure environment:
   - `GOOGLE_CLIENT_ID` (**required**)
   - `JWT_SECRET` (set a strong secret)
   - `DATABASE_URL` (default points to local postgres)
   - `FRONTEND_ORIGIN` (comma-separated allowed origins; empty = no cross-origin access)
   - `DB_MAX_CONNS` (default `30`)
3. Run the service:
   ```bash
   go run ./...
   ```

## Build and test
```bash
go test ./...
go build ./...
```

## Database
- SQL schema and indexes live in `database/migrations/`.
- Use the provided Makefile target:
  ```bash
  make migrate
  ```

## Security notes
- Passwords use `bcrypt` (`utils/password.go`), and verification uses `bcrypt.CompareHashAndPassword` (timing-safe comparison).
- CORS is deny-by-default unless `FRONTEND_ORIGIN` is explicitly configured.
- Extra rate limiting is applied to `/auth/login` and `/auth/register`.

## Documentation
- API guide: `docs/API.md`
- OpenAPI spec: `docs/openapi.yaml`
- Deployment guide: `docs/DEPLOYMENT.md`
- Elo model details: `docs/ELO.md`
