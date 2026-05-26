# QuizArena API Guide

Base URL: `http://localhost:8080`

## Authentication
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/google`
- `GET /auth/me` (JWT auth required)

## Quiz flow
1. `POST /quiz/start` with `{ "subject": "Algorithms" }`
2. `GET /quiz/next-question?subject=Algorithms`
3. `POST /quiz/submit-answer`
4. `GET /quiz/session/{id}` returns session + answered-question history

## Leaderboard
- `GET /leaderboard?limit=100&cursor={elo}:{user_id}`
- `GET /leaderboard/subject/{subject}?limit=100&cursor={elo}:{user_id}`
- `GET /leaderboard/user/{id}` returns user and DB-ranked rank.

Responses for list endpoints use:
```json
{
  "items": [...],
  "next_cursor": "1234:56"
}
```
`next_cursor` is omitted when there are no more records.

## Admin
Requires JWT + admin email configured in `ADMIN_EMAILS`.
- `POST /admin/questions`
- `PUT /admin/questions/{id}`
- `GET /admin/questions`
- `DELETE /admin/questions/{id}`

## Errors
All errors return:
```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "invalid payload"
  }
}
```
