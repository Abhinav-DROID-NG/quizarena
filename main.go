package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"quizarena/internal/quiz"
)

type app struct {
	db *sql.DB
}

type createUserRequest struct {
	Username string `json:"username"`
}

type createUserResponse struct {
	ID         int64   `json:"id"`
	Username   string  `json:"username"`
	CurrentElo float64 `json:"current_elo"`
	PeakElo    float64 `json:"peak_elo"`
	RankTier   string  `json:"rank_tier"`
	CreatedAt  string  `json:"created_at"`
}

type question struct {
	ID                  int64           `json:"question_id"`
	Prompt              string          `json:"prompt"`
	Subject             string          `json:"subject"`
	Difficulty          string          `json:"difficulty"`
	QuestionElo         float64         `json:"question_elo"`
	ExpectedTimeSeconds float64         `json:"expected_time_seconds"`
	Options             []string        `json:"options"`
	CorrectAnswer       string          `json:"-"`
	RawOptions          json.RawMessage `json:"-"`
}

type submitAnswerRequest struct {
	UserID           int64   `json:"user_id"`
	QuestionID       int64   `json:"question_id"`
	SelectedAnswer   string  `json:"selected_answer"`
	TimeTakenSeconds float64 `json:"time_taken_seconds"`
	Skipped          bool    `json:"skipped"`
}

type submitAnswerResponse struct {
	Correct                bool    `json:"correct"`
	CorrectAnswer          string  `json:"correct_answer"`
	TimeTaken              float64 `json:"time_taken"`
	TimeScore              float64 `json:"time_score"`
	PerformanceScore       float64 `json:"performance_score"`
	EloChange              int     `json:"elo_change"`
	NewUserElo             int     `json:"new_user_elo"`
	NextQuestionDifficulty string  `json:"next_question_difficulty"`
}

type leaderboardEntry struct {
	UserID               int64   `json:"user_id"`
	Username             string  `json:"username"`
	CurrentElo           int     `json:"current_elo"`
	PeakElo              int     `json:"peak_elo"`
	AccuracyPercentage   float64 `json:"accuracy_percentage"`
	AverageResponseTime  float64 `json:"average_response_time"`
	TotalQuestionsSolved int     `json:"total_questions_solved"`
	StrongestSubject     string  `json:"strongest_subject"`
	WeakestSubject       string  `json:"weakest_subject"`
	RankTier             string  `json:"rank_tier"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/quizarena?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	if err := initSchema(db); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	if err := seedQuestions(db); err != nil {
		log.Fatalf("seed questions: %v", err)
	}

	a := &app{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/users", a.handleCreateUser)
	mux.HandleFunc("/quiz/next", a.handleNextQuestion)
	mux.HandleFunc("/quiz/submit", a.handleSubmitAnswer)
	mux.HandleFunc("/leaderboard", a.handleLeaderboard)

	addr := ":8080"
	log.Printf("QuizArena backend listening on %s", addr)
	if err := http.ListenAndServe(addr, withJSONContentType(mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func withJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *app) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	var resp createUserResponse
	var createdAt time.Time
	err := a.db.QueryRow(`
		INSERT INTO users (username, current_elo, peak_elo, adaptation_aggressiveness)
		VALUES ($1, 1200, 1200, 1.0)
		RETURNING id, username, current_elo, peak_elo, created_at
	`, req.Username).Scan(&resp.ID, &resp.Username, &resp.CurrentElo, &resp.PeakElo, &createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "username already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create user"})
		return
	}
	resp.RankTier = string(quiz.RankForElo(resp.CurrentElo))
	resp.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusCreated, resp)
}

func (a *app) handleNextQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid user_id is required"})
		return
	}
	subject := strings.TrimSpace(r.URL.Query().Get("subject"))

	q, err := a.selectQuestionForUser(userID, subject)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, q)
}

func (a *app) handleSubmitAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req submitAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.UserID <= 0 || req.QuestionID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and question_id are required"})
		return
	}
	if req.TimeTakenSeconds < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "time_taken_seconds cannot be negative"})
		return
	}

	resp, err := a.evaluateAnswer(req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *app) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	rows, err := a.db.Query(`
SELECT id, username, current_elo, peak_elo, total_questions_solved, total_correct_answers, total_response_time
FROM users
ORDER BY peak_elo DESC, current_elo DESC
LIMIT $1
`, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read leaderboard"})
		return
	}
	defer rows.Close()

	entries := make([]leaderboardEntry, 0, limit)
	for rows.Next() {
		var e leaderboardEntry
		var currentElo float64
		var peakElo float64
		var totalCorrect int
		var totalTime float64
		if err := rows.Scan(&e.UserID, &e.Username, &currentElo, &peakElo, &e.TotalQuestionsSolved, &totalCorrect, &totalTime); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to parse leaderboard row"})
			return
		}
		e.CurrentElo = int(math.Round(currentElo))
		e.PeakElo = int(math.Round(peakElo))

		if e.TotalQuestionsSolved > 0 {
			e.AccuracyPercentage = math.Round((float64(totalCorrect)/float64(e.TotalQuestionsSolved))*10000) / 100
			e.AverageResponseTime = math.Round((totalTime/float64(e.TotalQuestionsSolved))*100) / 100
		}
		e.RankTier = string(quiz.RankForElo(float64(e.CurrentElo)))
		e.StrongestSubject, e.WeakestSubject, _ = a.subjectExtremes(e.UserID)
		entries = append(entries, e)
	}

	writeJSON(w, http.StatusOK, map[string]any{"leaderboard": entries})
}

func (a *app) evaluateAnswer(req submitAnswerRequest) (submitAnswerResponse, error) {
	tx, err := a.db.Begin()
	if err != nil {
		return submitAnswerResponse{}, err
	}
	defer tx.Rollback()

	var userElo, peakElo, adaptation float64
	var consecutiveSkips, guessStreak, totalSolved, totalCorrect int
	var totalResponseTime float64
	err = tx.QueryRow(`
SELECT current_elo, peak_elo, consecutive_skips, guess_streak, adaptation_aggressiveness,
       total_questions_solved, total_correct_answers, total_response_time
FROM users WHERE id=$1 FOR UPDATE
`, req.UserID).Scan(&userElo, &peakElo, &consecutiveSkips, &guessStreak, &adaptation, &totalSolved, &totalCorrect, &totalResponseTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return submitAnswerResponse{}, fmt.Errorf("user not found")
		}
		return submitAnswerResponse{}, err
	}

	var q question
	err = tx.QueryRow(`
SELECT id, prompt, subject, difficulty, question_elo, expected_time_seconds, options, correct_answer
FROM questions WHERE id=$1
`, req.QuestionID).Scan(&q.ID, &q.Prompt, &q.Subject, &q.Difficulty, &q.QuestionElo, &q.ExpectedTimeSeconds, &q.RawOptions, &q.CorrectAnswer)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return submitAnswerResponse{}, fmt.Errorf("question not found")
		}
		return submitAnswerResponse{}, err
	}

	correct := strings.EqualFold(strings.TrimSpace(req.SelectedAnswer), strings.TrimSpace(q.CorrectAnswer)) && !req.Skipped
	timeScore := quiz.TimeScore(req.TimeTakenSeconds, q.ExpectedTimeSeconds)
	basePerformance := quiz.PerformanceScore(req.TimeTakenSeconds, q.ExpectedTimeSeconds, correct)

	nextConsecutiveSkips := 0
	nextGuessStreak := 0
	if req.Skipped {
		nextConsecutiveSkips = consecutiveSkips + 1
	} else if !correct {
		nextConsecutiveSkips = 0
	}

	if !correct && q.ExpectedTimeSeconds > 0 && req.TimeTakenSeconds < (q.ExpectedTimeSeconds*0.5) {
		nextGuessStreak = guessStreak + 1
	}

	performance := quiz.ApplyPenalties(basePerformance, req.TimeTakenSeconds, q.ExpectedTimeSeconds, correct, req.Skipped, nextConsecutiveSkips, nextGuessStreak)
	newElo, eloChange := quiz.UpdateElo(userElo, q.QuestionElo, quiz.Difficulty(strings.ToLower(q.Difficulty)), performance)

	if req.Skipped {
		adaptation = math.Max(0.5, adaptation-0.1)
	} else {
		adaptation = math.Min(1.0, adaptation+0.05)
	}

	if newElo > peakElo {
		peakElo = newElo
	}

	totalSolved++
	if correct {
		totalCorrect++
	}
	totalResponseTime += req.TimeTakenSeconds

	if _, err = tx.Exec(`
UPDATE users
SET current_elo=$1, peak_elo=$2, consecutive_skips=$3, guess_streak=$4,
    adaptation_aggressiveness=$5, total_questions_solved=$6,
    total_correct_answers=$7, total_response_time=$8
WHERE id=$9
`, newElo, peakElo, nextConsecutiveSkips, nextGuessStreak, adaptation, totalSolved, totalCorrect, totalResponseTime, req.UserID); err != nil {
		return submitAnswerResponse{}, err
	}

	if _, err = tx.Exec(`
INSERT INTO attempts (user_id, question_id, selected_answer, time_taken_seconds, correct, performance_score, elo_change, skipped)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`, req.UserID, req.QuestionID, req.SelectedAnswer, req.TimeTakenSeconds, correct, performance, eloChange, req.Skipped); err != nil {
		return submitAnswerResponse{}, err
	}

	if _, err = tx.Exec(`
INSERT INTO user_subject_stats (user_id, subject, attempts, correct)
VALUES ($1,$2,1,$3)
ON CONFLICT (user_id, subject)
DO UPDATE SET
  attempts = user_subject_stats.attempts + 1,
  correct = user_subject_stats.correct + EXCLUDED.correct
`, req.UserID, q.Subject, boolToInt(correct)); err != nil {
		return submitAnswerResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return submitAnswerResponse{}, err
	}

	nextQ, _ := a.selectQuestionForUser(req.UserID, q.Subject)
	nextDifficulty := "medium"
	if nextQ != nil {
		nextDifficulty = nextQ.Difficulty
	}

	return submitAnswerResponse{
		Correct:                correct,
		CorrectAnswer:          q.CorrectAnswer,
		TimeTaken:              req.TimeTakenSeconds,
		TimeScore:              roundTo(timeScore, 2),
		PerformanceScore:       roundTo(performance, 2),
		EloChange:              eloChange,
		NewUserElo:             int(math.Round(newElo)),
		NextQuestionDifficulty: nextDifficulty,
	}, nil
}

func (a *app) selectQuestionForUser(userID int64, subject string) (*question, error) {
	var userElo, adaptation float64
	err := a.db.QueryRow("SELECT current_elo, adaptation_aggressiveness FROM users WHERE id=$1", userID).Scan(&userElo, &adaptation)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	targetElo := userElo
	band := 50.0 * adaptation
	r := rand.Float64()
	switch {
	case r < 0.70:
		targetElo += rand.Float64()*(2*band) - band
	case r < 0.90:
		targetElo += 25 + rand.Float64()*50
	default:
		targetElo -= 25 + rand.Float64()*50
	}

	query := `
SELECT id, prompt, subject, difficulty, question_elo, expected_time_seconds, options, correct_answer
FROM questions
WHERE ($1 = '' OR subject = $1)
ORDER BY ABS(question_elo - $2), RANDOM()
LIMIT 1
`
	var q question
	err = a.db.QueryRow(query, subject, targetElo).Scan(&q.ID, &q.Prompt, &q.Subject, &q.Difficulty, &q.QuestionElo, &q.ExpectedTimeSeconds, &q.RawOptions, &q.CorrectAnswer)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if err := json.Unmarshal(q.RawOptions, &q.Options); err != nil {
		return nil, err
	}
	return &q, nil
}

func (a *app) subjectExtremes(userID int64) (strongest, weakest string, err error) {
	rows, err := a.db.Query(`
SELECT subject, attempts, correct
FROM user_subject_stats
WHERE user_id=$1
`, userID)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	bestRate := -1.0
	worstRate := 2.0
	for rows.Next() {
		var subject string
		var attempts, correct int
		if err := rows.Scan(&subject, &attempts, &correct); err != nil {
			return "", "", err
		}
		if attempts == 0 {
			continue
		}
		rate := float64(correct) / float64(attempts)
		if rate > bestRate {
			bestRate = rate
			strongest = subject
		}
		if rate < worstRate {
			worstRate = rate
			weakest = subject
		}
	}
	if strongest == "" {
		strongest = "n/a"
	}
	if weakest == "" {
		weakest = "n/a"
	}
	return strongest, weakest, nil
}

func initSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
id BIGSERIAL PRIMARY KEY,
username TEXT UNIQUE NOT NULL,
current_elo DOUBLE PRECISION NOT NULL DEFAULT 1200,
peak_elo DOUBLE PRECISION NOT NULL DEFAULT 1200,
consecutive_skips INTEGER NOT NULL DEFAULT 0,
guess_streak INTEGER NOT NULL DEFAULT 0,
adaptation_aggressiveness DOUBLE PRECISION NOT NULL DEFAULT 1.0,
total_questions_solved INTEGER NOT NULL DEFAULT 0,
total_correct_answers INTEGER NOT NULL DEFAULT 0,
total_response_time DOUBLE PRECISION NOT NULL DEFAULT 0,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
		`CREATE TABLE IF NOT EXISTS questions (
id BIGSERIAL PRIMARY KEY,
prompt TEXT NOT NULL,
subject TEXT NOT NULL,
difficulty TEXT NOT NULL CHECK (difficulty IN ('easy','medium','hard')),
question_elo DOUBLE PRECISION NOT NULL,
expected_time_seconds DOUBLE PRECISION NOT NULL,
options JSONB NOT NULL,
correct_answer TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS attempts (
id BIGSERIAL PRIMARY KEY,
user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
selected_answer TEXT,
time_taken_seconds DOUBLE PRECISION NOT NULL,
correct BOOLEAN NOT NULL,
performance_score DOUBLE PRECISION NOT NULL,
elo_change INTEGER NOT NULL,
skipped BOOLEAN NOT NULL DEFAULT FALSE,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
		`CREATE TABLE IF NOT EXISTS user_subject_stats (
user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
subject TEXT NOT NULL,
attempts INTEGER NOT NULL DEFAULT 0,
correct INTEGER NOT NULL DEFAULT 0,
PRIMARY KEY (user_id, subject)
)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func seedQuestions(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM questions`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	type seedQuestion struct {
		Prompt       string
		Subject      string
		Difficulty   string
		QuestionElo  float64
		ExpectedTime float64
		Options      []string
		Answer       string
	}

	seed := []seedQuestion{
		{"What is 2 + 2?", "math", "easy", 900, 12, []string{"3", "4", "5", "6"}, "4"},
		{"Derivative of x^2?", "math", "medium", 1250, 20, []string{"x", "2x", "x^2", "2"}, "2x"},
		{"Integral of 2x dx?", "math", "hard", 1700, 25, []string{"x^2 + C", "2x + C", "x + C", "4x + C"}, "x^2 + C"},
		{"Capital of France?", "geography", "easy", 900, 10, []string{"Paris", "Berlin", "Madrid", "Rome"}, "Paris"},
		{"Largest ocean?", "geography", "medium", 1300, 15, []string{"Atlantic", "Arctic", "Indian", "Pacific"}, "Pacific"},
		{"Mitochondria is the ___ of the cell.", "science", "easy", 950, 12, []string{"brain", "powerhouse", "wall", "nucleus"}, "powerhouse"},
		{"Speed of light (km/s)?", "science", "hard", 1750, 20, []string{"300", "3000", "300000", "3000000"}, "300000"},
	}

	for _, q := range seed {
		opt, err := json.Marshal(q.Options)
		if err != nil {
			return err
		}
		if _, err := db.Exec(`
INSERT INTO questions (prompt, subject, difficulty, question_elo, expected_time_seconds, options, correct_answer)
VALUES ($1,$2,$3,$4,$5,$6,$7)
`, q.Prompt, q.Subject, q.Difficulty, q.QuestionElo, q.ExpectedTime, opt, q.Answer); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func roundTo(val float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(val*factor) / factor
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
