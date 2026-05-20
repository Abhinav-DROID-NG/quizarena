# QuizArena Backend

Go + PostgreSQL backend for a single-player adaptive quiz system with Elo ranking.

## Features
- Elo updates for users per submission (difficulty-specific K-factor: easy=16, medium=24, hard=32)
- Adaptive question selection (70% near Elo, 20% harder, 10% easier)
- Performance scoring (`0.8 * time_score + 0.2 * correctness`)
- Anti-guessing penalties (very fast wrong answers, guess streaks, skip streaks)
- Leaderboard metrics (`current_elo`, `peak_elo`, `accuracy_percentage`, `average_response_time`, `total_questions_solved`, strongest/weakest subject)
- Rank tiers: Bronze, Silver, Gold, Platinum, Diamond, Master

## Requirements
- Go 1.22+
- PostgreSQL

## Run
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/quizarena?sslmode=disable"
go run .
```

The service starts on `:8080` and auto-creates schema + seeds starter questions.

## API

### `POST /users`
Request:
```json
{"username":"player1"}
```

### `GET /quiz/next?user_id=1&subject=math`
Returns the next adaptive question.

### `POST /quiz/submit`
Request:
```json
{
  "user_id": 1,
  "question_id": 2,
  "selected_answer": "2x",
  "time_taken_seconds": 12.4,
  "skipped": false
}
```
Response format:
```json
{
  "correct": true,
  "correct_answer": "2x",
  "time_taken": 12.4,
  "time_score": 0.38,
  "performance_score": 0.5,
  "elo_change": 12,
  "new_user_elo": 1212,
  "next_question_difficulty": "medium"
}
```

### `GET /leaderboard?limit=20`
Returns ranked players with leaderboard metrics and derived rank tier.
