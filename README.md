# QuizArena Backend

Production-oriented Go backend for adaptive quiz gameplay with Google OAuth2, JWT auth, PostgreSQL, and Redis caching.

## Features
- Google OAuth2 login endpoint + JWT issuance
- JWT middleware for protected endpoints
- Rate limiting (10 req/s per IP)
- CORS configuration for frontend origins
- Elo engine with:
  - Expected probability `1 / (1 + 10^((question_elo - user_elo)/400))`
  - Performance score (80% time + 20% correctness)
  - Difficulty K-factors (easy=16, medium=24, hard=32)
  - Anti-guessing penalties and confidence score
- PostgreSQL schema and indexes
- Redis leaderboard caching with graceful DB fallback
- Health check endpoint (`GET /health`)
- Graceful shutdown

## Run locally
1. Start dependencies:
   ```bash
   docker compose up -d
   ```
2. Set env vars (optional overrides):
   - `GOOGLE_CLIENT_ID`
   - `JWT_SECRET`
   - `DATABASE_URL`
   - `REDIS_ADDR`
   - `FRONTEND_ORIGIN`
3. Run server:
   ```bash
   go run ./...
   ```

## Test
```bash
go test ./...
```

## API docs
OpenAPI spec: `docs/openapi.yaml`
