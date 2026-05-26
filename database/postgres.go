package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/models"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	Pool *pgxpool.Pool
}

var ErrConnectionExhausted = errors.New("database connection pool exhausted")

func New(ctx context.Context, databaseURL string, maxConns int32) (*Client, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	if maxConns < utils.MinDBMaxConns || maxConns > utils.MaxDBMaxConns {
		return nil, fmt.Errorf("invalid max connections: %d", maxConns)
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = maxConns / 4
	if cfg.MinConns < 4 {
		cfg.MinConns = 4
	}
	if cfg.MinConns > cfg.MaxConns {
		cfg.MinConns = cfg.MaxConns
	}
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Client{Pool: pool}, nil
}

func (c *Client) Ping(ctx context.Context) error { return c.Pool.Ping(ctx) }
func (c *Client) Close()                         { c.Pool.Close() }

func (c *Client) IsConnectionExhausted() bool {
	if c == nil || c.Pool == nil {
		return false
	}
	stats := c.Pool.Stat()
	return stats.MaxConns() > 0 && stats.AcquiredConns() >= stats.MaxConns() && stats.IdleConns() == 0
}

func (c *Client) annotatePoolError(err error) error {
	if err == nil {
		return nil
	}
	if c.IsConnectionExhausted() {
		return fmt.Errorf("%w: %v", ErrConnectionExhausted, err)
	}
	return err
}

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
		return u, c.annotatePoolError(err)
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
		return u, c.annotatePoolError(err)
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
		return u, "", c.annotatePoolError(err)
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
		return u, c.annotatePoolError(err)
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
	return sessionID, c.annotatePoolError(err)
}

func (c *Client) GetSession(ctx context.Context, sessionID, userID int64) (models.QuizSession, error) {
	var s models.QuizSession
	err := c.Pool.QueryRow(ctx, `SELECT id, user_id, subject, status FROM quiz_sessions WHERE id = $1 AND user_id = $2`, sessionID, userID).Scan(&s.ID, &s.UserID, &s.Subject, &s.Status)
	return s, c.annotatePoolError(err)
}

func (c *Client) GetQuestionByID(ctx context.Context, questionID int64) (models.Question, error) {
	var q models.Question
	err := c.Pool.QueryRow(ctx, `SELECT id, subject, type, difficulty, question_text, options, correct_answers, question_elo, expected_time_seconds FROM questions WHERE id = $1`, questionID).Scan(
		&q.ID, &q.Subject, &q.Type, &q.Difficulty, &q.QuestionText, &q.Options, &q.CorrectAnswers, &q.QuestionElo, &q.ExpectedTimeSeconds,
	)
	return q, c.annotatePoolError(err)
}

func (c *Client) GetAdaptiveQuestion(ctx context.Context, subject string, targetElo int) (models.Question, error) {
	const q = `
WITH lower_band AS (
	SELECT id, subject, type, difficulty, question_text, options, correct_answers, question_elo, expected_time_seconds
	FROM questions
	WHERE ($1 = '' OR subject = $1) AND question_elo <= $2
	ORDER BY question_elo DESC, id DESC
	LIMIT 25
),
upper_band AS (
	SELECT id, subject, type, difficulty, question_text, options, correct_answers, question_elo, expected_time_seconds
	FROM questions
	WHERE ($1 = '' OR subject = $1) AND question_elo >= $2
	ORDER BY question_elo ASC, id ASC
	LIMIT 25
),
candidates AS (
	SELECT * FROM lower_band
	UNION ALL
	SELECT * FROM upper_band
)
SELECT id, subject, type, difficulty, question_text, options, correct_answers, question_elo, expected_time_seconds
FROM candidates
ORDER BY ABS(question_elo - $2), random()
LIMIT 1`
	var question models.Question
	err := c.Pool.QueryRow(ctx, q, subject, targetElo).Scan(
		&question.ID, &question.Subject, &question.Type, &question.Difficulty, &question.QuestionText, &question.Options, &question.CorrectAnswers, &question.QuestionElo, &question.ExpectedTimeSeconds,
	)
	return question, c.annotatePoolError(err)
}

func (c *Client) SaveAnswerAndUpdateStats(ctx context.Context, sessionID, questionID, userID int64, selectedAnswers, correctAnswers []string, timeTaken, timeScore, performance float64, eloChange, newElo int) error {
	// Correctness check logic is moved to handler for more flexibility, or kept here if simple
	correct := true
	if len(selectedAnswers) != len(correctAnswers) {
		correct = false
	} else {
		// Simple set comparison (assuming sorted or small enough)
		m := make(map[string]bool)
		for _, v := range correctAnswers {
			m[v] = true
		}
		for _, v := range selectedAnswers {
			if !m[v] {
				correct = false
				break
			}
		}
	}

	tx, err := c.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return c.annotatePoolError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	insertCT, err := tx.Exec(ctx, `INSERT INTO quiz_answers
(session_id, question_id, user_id, selected_answers, correct, time_taken_seconds, time_score, performance_score, elo_change)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		sessionID, questionID, userID, selectedAnswers, correct, timeTaken, timeScore, performance, eloChange,
	)
	if err != nil {
		return err
	}
	if insertCT.RowsAffected() != 1 {
		return errors.New("failed to persist answer")
	}

	updateCT, err := tx.Exec(ctx, `UPDATE users
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
	if updateCT.RowsAffected() != 1 {
		return errors.New("failed to update user stats")
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Client) ListLeaderboard(ctx context.Context, subject string, limit int) ([]models.User, error) {
	users, _, _, _, err := c.ListLeaderboardPage(ctx, subject, limit, 0, 0)
	return users, err
}

func (c *Client) ListLeaderboardPage(ctx context.Context, subject string, limit int, cursorElo int, cursorUserID int64) ([]models.User, int, int64, bool, error) {
	if limit <= 0 {
		limit = 100
	}
	fetchLimit := limit + 1
	q := `SELECT id, google_sub, email, name, picture, current_elo, peak_elo, accuracy_percentage, average_response_time,
total_questions_solved, strongest_subject, weakest_subject
FROM users
WHERE ($2 = 0 AND $3 = 0) OR (current_elo < $2 OR (current_elo = $2 AND id < $3))
ORDER BY current_elo DESC, id DESC
LIMIT $1`
	var rows pgx.Rows
	var err error
	if subject != "" {
		q = `SELECT u.id, u.google_sub, u.email, u.name, u.picture, u.current_elo, u.peak_elo, u.accuracy_percentage, u.average_response_time,
u.total_questions_solved, u.strongest_subject, u.weakest_subject
FROM users u
WHERE EXISTS (
	SELECT 1 FROM quiz_sessions s
	WHERE s.user_id = u.id AND s.subject = $2
)
AND (($3 = 0 AND $4 = 0) OR (u.current_elo < $3 OR (u.current_elo = $3 AND u.id < $4)))
ORDER BY u.current_elo DESC, u.id DESC
LIMIT $1`
		rows, err = c.Pool.Query(ctx, q, fetchLimit, subject, cursorElo, cursorUserID)
	} else {
		rows, err = c.Pool.Query(ctx, q, fetchLimit, cursorElo, cursorUserID)
	}
	if err != nil {
		return nil, 0, 0, false, c.annotatePoolError(err)
	}
	defer rows.Close()

	users := make([]models.User, 0, fetchLimit)
	for rows.Next() {
		var u models.User
		var gSub, pic *string
		if err := rows.Scan(&u.ID, &gSub, &u.Email, &u.Name, &pic, &u.CurrentElo, &u.PeakElo,
			&u.AccuracyPercentage, &u.AverageResponseTime, &u.TotalQuestions, &u.StrongestSubject, &u.WeakestSubject); err != nil {
			return nil, 0, 0, false, err
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
		return nil, 0, 0, false, rows.Err()
	}
	if len(users) == 0 {
		return []models.User{}, 0, 0, false, nil
	}
	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}
	nextCursorElo := 0
	var nextCursorUserID int64
	if hasMore {
		last := users[len(users)-1]
		nextCursorElo = last.CurrentElo
		nextCursorUserID = last.ID
	}
	return users, nextCursorElo, nextCursorUserID, hasMore, nil
}

func (c *Client) GetUserRank(ctx context.Context, userID int64) (int, error) {
	const q = `
SELECT ranked.rank
FROM (
	SELECT id, ROW_NUMBER() OVER (ORDER BY current_elo DESC, id DESC) AS rank
	FROM users
) AS ranked
WHERE ranked.id = $1`
	var rank int
	err := c.Pool.QueryRow(ctx, q, userID).Scan(&rank)
	if err != nil {
		return 0, c.annotatePoolError(err)
	}
	return rank, nil
}

func (c *Client) GetSessionHistory(ctx context.Context, sessionID, userID int64) (models.QuizSession, []models.SessionAnswerHistory, error) {
	session, err := c.GetSession(ctx, sessionID, userID)
	if err != nil {
		return models.QuizSession{}, nil, err
	}

	const q = `SELECT a.id, a.question_id, q.question_text, q.options, a.selected_answers, q.correct_answers,
	a.correct, a.time_taken_seconds, a.time_score, a.performance_score, a.elo_change, a.created_at
FROM quiz_answers a
JOIN questions q ON q.id = a.question_id
WHERE a.session_id = $1 AND a.user_id = $2
ORDER BY a.created_at ASC, a.id ASC`
	rows, err := c.Pool.Query(ctx, q, sessionID, userID)
	if err != nil {
		return models.QuizSession{}, nil, c.annotatePoolError(err)
	}
	defer rows.Close()

	history := make([]models.SessionAnswerHistory, 0)
	for rows.Next() {
		var answer models.SessionAnswerHistory
		if err := rows.Scan(
			&answer.AnswerID,
			&answer.QuestionID,
			&answer.QuestionText,
			&answer.Options,
			&answer.SelectedAnswers,
			&answer.CorrectAnswers,
			&answer.Correct,
			&answer.TimeTakenSeconds,
			&answer.TimeScore,
			&answer.PerformanceScore,
			&answer.EloChange,
			&answer.AnsweredAt,
		); err != nil {
			return models.QuizSession{}, nil, c.annotatePoolError(err)
		}
		history = append(history, answer)
	}
	if err := rows.Err(); err != nil {
		return models.QuizSession{}, nil, c.annotatePoolError(err)
	}
	return session, history, nil
}

func (c *Client) UpsertQuestion(ctx context.Context, q models.Question) (int64, error) {
	if q.ID == 0 {
		if err := utils.ValidateQuestionInput(q); err != nil {
			return 0, err
		}
		var id int64
		err := c.Pool.QueryRow(ctx, `INSERT INTO questions (subject, type, difficulty, question_text, options, correct_answers, question_elo, expected_time_seconds)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, q.Subject, q.Type, q.Difficulty, q.QuestionText, q.Options, q.CorrectAnswers, q.QuestionElo, q.ExpectedTimeSeconds).Scan(&id)
		return id, c.annotatePoolError(err)
	}
	if err := utils.ValidateQuestionInput(q); err != nil {
		return 0, err
	}
	ct, err := c.Pool.Exec(ctx, `UPDATE questions SET subject=$1,type=$2,difficulty=$3,question_text=$4,options=$5,correct_answers=$6,question_elo=$7,expected_time_seconds=$8,updated_at=NOW() WHERE id=$9`,
		q.Subject, q.Type, q.Difficulty, q.QuestionText, q.Options, q.CorrectAnswers, q.QuestionElo, q.ExpectedTimeSeconds, q.ID)
	if err != nil {
		return 0, c.annotatePoolError(err)
	}
	if ct.RowsAffected() == 0 {
		return 0, errors.New("question not found")
	}
	return q.ID, nil
}

func (c *Client) ListQuestions(ctx context.Context, limit int) ([]models.Question, error) {
	rows, err := c.Pool.Query(ctx, `SELECT id, subject, type, difficulty, question_text, options, correct_answers, question_elo, expected_time_seconds FROM questions ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, c.annotatePoolError(err)
	}
	defer rows.Close()
	questions := []models.Question{}
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(&q.ID, &q.Subject, &q.Type, &q.Difficulty, &q.QuestionText, &q.Options, &q.CorrectAnswers, &q.QuestionElo, &q.ExpectedTimeSeconds); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

func (c *Client) DeleteQuestion(ctx context.Context, id int64) error {
	ct, err := c.Pool.Exec(ctx, `DELETE FROM questions WHERE id=$1`, id)
	if err != nil {
		return c.annotatePoolError(err)
	}
	if ct.RowsAffected() == 0 {
		return errors.New("question not found")
	}
	return nil
}

func (c *Client) ListSubjects(ctx context.Context) ([]string, error) {
	rows, err := c.Pool.Query(ctx, `SELECT DISTINCT subject FROM questions ORDER BY subject`)
	if err != nil {
		return nil, c.annotatePoolError(err)
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

type GlobalStats struct {
	TotalQuestions int     `json:"total_questions"`
	TotalUsers     int     `json:"total_users"`
	TotalSubjects  int     `json:"total_subjects"`
	SuccessRate    float64 `json:"success_rate"`
}

func (c *Client) GetGlobalStats(ctx context.Context) (GlobalStats, error) {
	const q = `
SELECT
    (SELECT COUNT(*) FROM questions) as total_questions,
    (SELECT COUNT(*) FROM users) as total_users,
    (SELECT COUNT(DISTINCT subject) FROM questions) as total_subjects,
    COALESCE((SELECT AVG(accuracy_percentage) FROM users WHERE total_questions_solved > 0), 0) as success_rate
`
	var stats GlobalStats
	err := c.Pool.QueryRow(ctx, q).Scan(&stats.TotalQuestions, &stats.TotalUsers, &stats.TotalSubjects, &stats.SuccessRate)
	return stats, c.annotatePoolError(err)
}

type SubjectWithCount struct {
	Subject       string `json:"subject"`
	QuestionCount int    `json:"question_count"`
}

func (c *Client) ListSubjectsWithCounts(ctx context.Context) ([]SubjectWithCount, error) {
	const q = `SELECT subject, COUNT(*) as question_count FROM questions GROUP BY subject ORDER BY subject`
	rows, err := c.Pool.Query(ctx, q)
	if err != nil {
		return nil, c.annotatePoolError(err)
	}
	defer rows.Close()
	var subjects []SubjectWithCount
	for rows.Next() {
		var s SubjectWithCount
		if err := rows.Scan(&s.Subject, &s.QuestionCount); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}
	return subjects, rows.Err()
}
