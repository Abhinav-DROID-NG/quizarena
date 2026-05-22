package database

import (
	"context"
	"errors"

	"github.com/Abhinav-DROID-NG/quizarena/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string, maxConns int32) (*Client, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Client{Pool: pool}, nil
}

func (c *Client) Ping(ctx context.Context) error { return c.Pool.Ping(ctx) }
func (c *Client) Close()                         { c.Pool.Close() }

func (c *Client) UpsertOAuthUser(ctx context.Context, sub, email, name, picture string) (models.User, error) {
	const q = `
INSERT INTO users (google_sub, email, name, picture)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email)
DO UPDATE SET google_sub = COALESCE(users.google_sub, EXCLUDED.google_sub),
              name = EXCLUDED.name,
              picture = COALESCE(users.picture, EXCLUDED.picture),
              updated_at = NOW()
RETURNING id, google_sub, email, name, picture, current_elo, peak_elo,
accuracy_percentage, average_response_time, total_questions_solved, strongest_subject, weakest_subject`
	var u models.User
	var gSub, pic *string
	err := c.Pool.QueryRow(ctx, q, sub, email, name, picture).Scan(
		&u.ID, &gSub, &u.Email, &u.Name, &pic, &u.CurrentElo, &u.PeakElo,
		&u.AccuracyPercentage, &u.AverageResponseTime, &u.TotalQuestions, &u.StrongestSubject, &u.WeakestSubject,
	)
	if err != nil {
		return u, err
	}
	if gSub != nil {
		u.GoogleSub = *gSub
	}
	if pic != nil {
		u.Picture = *pic
	}
	return u, nil
}

func (c *Client) CreateUser(ctx context.Context, email, passwordHash, name string) (models.User, error) {
	const q = `
INSERT INTO users (email, password_hash, name)
VALUES ($1, $2, $3)
RETURNING id, google_sub, email, name, picture, current_elo, peak_elo,
accuracy_percentage, average_response_time, total_questions_solved, strongest_subject, weakest_subject`
	var u models.User
	var gSub, pic *string
	err := c.Pool.QueryRow(ctx, q, email, passwordHash, name).Scan(
		&u.ID, &gSub, &u.Email, &u.Name, &pic, &u.CurrentElo, &u.PeakElo,
		&u.AccuracyPercentage, &u.AverageResponseTime, &u.TotalQuestions, &u.StrongestSubject, &u.WeakestSubject,
	)
	if err != nil {
		return u, err
	}
	if gSub != nil {
		u.GoogleSub = *gSub
	}
	if pic != nil {
		u.Picture = *pic
	}
	return u, nil
}

func (c *Client) GetUserByEmail(ctx context.Context, email string) (models.User, string, error) {
	const q = `SELECT id, google_sub, email, name, picture, current_elo, peak_elo,
accuracy_percentage, average_response_time, total_questions_solved, strongest_subject, weakest_subject, password_hash
FROM users WHERE email = $1`
	var u models.User
	var gSub, picture, pwHash *string
	err := c.Pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &gSub, &u.Email, &u.Name, &picture, &u.CurrentElo, &u.PeakElo,
		&u.AccuracyPercentage, &u.AverageResponseTime, &u.TotalQuestions, &u.StrongestSubject, &u.WeakestSubject, &pwHash,
	)
	if err != nil {
		return u, "", err
	}
	if gSub != nil {
		u.GoogleSub = *gSub
	}
	if picture != nil {
		u.Picture = *picture
	}
	hash := ""
	if pwHash != nil {
		hash = *pwHash
	}
	return u, hash, nil
}

func (c *Client) GetUserByID(ctx context.Context, userID int64) (models.User, error) {
	const q = `SELECT id, google_sub, email, name, picture, current_elo, peak_elo,
accuracy_percentage, average_response_time, total_questions_solved, strongest_subject, weakest_subject
FROM users WHERE id = $1`
	var u models.User
	var gSub, picture *string
	err := c.Pool.QueryRow(ctx, q, userID).Scan(
		&u.ID, &gSub, &u.Email, &u.Name, &picture, &u.CurrentElo, &u.PeakElo,
		&u.AccuracyPercentage, &u.AverageResponseTime, &u.TotalQuestions, &u.StrongestSubject, &u.WeakestSubject,
	)
	if err != nil {
		return u, err
	}
	if gSub != nil {
		u.GoogleSub = *gSub
	}
	if picture != nil {
		u.Picture = *picture
	}
	return u, nil
}

func (c *Client) CreateSession(ctx context.Context, userID int64, subject string) (int64, error) {
	var sessionID int64
	err := c.Pool.QueryRow(ctx, `INSERT INTO quiz_sessions (user_id, subject, status) VALUES ($1, $2, 'active') RETURNING id`, userID, subject).Scan(&sessionID)
	return sessionID, err
}

func (c *Client) GetSession(ctx context.Context, sessionID, userID int64) (models.QuizSession, error) {
	var s models.QuizSession
	err := c.Pool.QueryRow(ctx, `SELECT id, user_id, subject, status FROM quiz_sessions WHERE id = $1 AND user_id = $2`, sessionID, userID).Scan(&s.ID, &s.UserID, &s.Subject, &s.Status)
	return s, err
}

func (c *Client) GetQuestionByID(ctx context.Context, questionID int64) (models.Question, error) {
	var q models.Question
	err := c.Pool.QueryRow(ctx, `SELECT id, subject, difficulty, question_text, options, correct_answer, question_elo, expected_time_seconds FROM questions WHERE id = $1`, questionID).Scan(
		&q.ID, &q.Subject, &q.Difficulty, &q.QuestionText, &q.Options, &q.CorrectAnswer, &q.QuestionElo, &q.ExpectedTimeSeconds,
	)
	return q, err
}

func (c *Client) GetAdaptiveQuestion(ctx context.Context, subject string, targetElo int) (models.Question, error) {
	const q = `SELECT id, subject, difficulty, question_text, options, correct_answer, question_elo, expected_time_seconds
FROM questions
WHERE ($1 = '' OR subject = $1)
ORDER BY ABS(question_elo - $2), random()
LIMIT 1`
	var question models.Question
	err := c.Pool.QueryRow(ctx, q, subject, targetElo).Scan(
		&question.ID, &question.Subject, &question.Difficulty, &question.QuestionText, &question.Options, &question.CorrectAnswer, &question.QuestionElo, &question.ExpectedTimeSeconds,
	)
	return question, err
}

func (c *Client) SaveAnswerAndUpdateStats(ctx context.Context, sessionID, questionID, userID int64, selectedAnswer, correctAnswer string, timeTaken, timeScore, performance float64, eloChange, newElo int) error {
	correct := selectedAnswer == correctAnswer
	tx, err := c.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `INSERT INTO quiz_answers
(session_id, question_id, user_id, selected_answer, correct, time_taken_seconds, time_score, performance_score, elo_change)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		sessionID, questionID, userID, selectedAnswer, correct, timeTaken, timeScore, performance, eloChange,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE users
SET current_elo = $1,
peak_elo = GREATEST(peak_elo, $1),
total_questions_solved = total_questions_solved + 1,
accuracy_percentage = CASE WHEN total_questions_solved = 0 THEN CASE WHEN $2 THEN 100 ELSE 0 END
ELSE (((accuracy_percentage * total_questions_solved) + CASE WHEN $2 THEN 100 ELSE 0 END)/(total_questions_solved + 1)) END,
average_response_time = CASE WHEN total_questions_solved = 0 THEN $3
ELSE ((average_response_time * total_questions_solved) + $3)/(total_questions_solved + 1) END,
updated_at = NOW()
WHERE id = $4`, newElo, correct, timeTaken, userID)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Client) ListLeaderboard(ctx context.Context, subject string, limit int) ([]models.User, error) {
	q := `SELECT id, google_sub, email, name, picture, current_elo, peak_elo, accuracy_percentage, average_response_time,
total_questions_solved, strongest_subject, weakest_subject FROM users ORDER BY current_elo DESC LIMIT $1`
	if subject != "" {
		q = `SELECT u.id, u.google_sub, u.email, u.name, u.picture, u.current_elo, u.peak_elo, u.accuracy_percentage, u.average_response_time,
u.total_questions_solved, u.strongest_subject, u.weakest_subject
FROM users u
JOIN quiz_sessions s ON s.user_id = u.id AND s.subject = $1
GROUP BY u.id
ORDER BY u.current_elo DESC LIMIT $2`
	}

	var rows pgx.Rows
	var err error
	if subject != "" {
		rows, err = c.Pool.Query(ctx, q, subject, limit)
	} else {
		rows, err = c.Pool.Query(ctx, q, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0, limit)
	for rows.Next() {
		var u models.User
		var gSub, pic *string
		if err := rows.Scan(&u.ID, &gSub, &u.Email, &u.Name, &pic, &u.CurrentElo, &u.PeakElo,
			&u.AccuracyPercentage, &u.AverageResponseTime, &u.TotalQuestions, &u.StrongestSubject, &u.WeakestSubject); err != nil {
			return nil, err
		}
		if gSub != nil {
			u.GoogleSub = *gSub
		}
		if pic != nil {
			u.Picture = *pic
		}
		users = append(users, u)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if len(users) == 0 {
		return []models.User{}, nil
	}
	return users, nil
}

func (c *Client) UpsertQuestion(ctx context.Context, q models.Question) (int64, error) {
	if q.ID == 0 {
		var id int64
		err := c.Pool.QueryRow(ctx, `INSERT INTO questions (subject, difficulty, question_text, options, correct_answer, question_elo, expected_time_seconds)
VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, q.Subject, q.Difficulty, q.QuestionText, q.Options, q.CorrectAnswer, q.QuestionElo, q.ExpectedTimeSeconds).Scan(&id)
		return id, err
	}
	ct, err := c.Pool.Exec(ctx, `UPDATE questions SET subject=$1,difficulty=$2,question_text=$3,options=$4,correct_answer=$5,question_elo=$6,expected_time_seconds=$7,updated_at=NOW() WHERE id=$8`,
		q.Subject, q.Difficulty, q.QuestionText, q.Options, q.CorrectAnswer, q.QuestionElo, q.ExpectedTimeSeconds, q.ID)
	if err != nil {
		return 0, err
	}
	if ct.RowsAffected() == 0 {
		return 0, errors.New("question not found")
	}
	return q.ID, nil
}

func (c *Client) ListQuestions(ctx context.Context, limit int) ([]models.Question, error) {
	rows, err := c.Pool.Query(ctx, `SELECT id, subject, difficulty, question_text, options, correct_answer, question_elo, expected_time_seconds FROM questions ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	questions := []models.Question{}
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(&q.ID, &q.Subject, &q.Difficulty, &q.QuestionText, &q.Options, &q.CorrectAnswer, &q.QuestionElo, &q.ExpectedTimeSeconds); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

func (c *Client) DeleteQuestion(ctx context.Context, id int64) error {
	ct, err := c.Pool.Exec(ctx, `DELETE FROM questions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New("question not found")
	}
	return nil
}

func (c *Client) ListSubjects(ctx context.Context) ([]string, error) {
	rows, err := c.Pool.Query(ctx, `SELECT DISTINCT subject FROM questions ORDER BY subject`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subjects []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}
	return subjects, rows.Err()
}
