package models

import "time"

type QuizSession struct {
	ID      int64  `json:"id"`
	UserID  int64  `json:"user_id"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
}

type AnswerResponse struct {
	Correct                bool     `json:"correct"`
	CorrectAnswers         []string `json:"correct_answers"`
	TimeTaken              float64  `json:"time_taken"`
	TimeScore              float64  `json:"time_score"`
	PerformanceScore       float64  `json:"performance_score"`
	EloChange              int      `json:"elo_change"`
	NewUserElo             int      `json:"new_user_elo"`
	NextQuestionDifficulty string   `json:"next_question_difficulty"`
	ConfidenceScore        float64  `json:"confidence_score"`
}

type SessionAnswerHistory struct {
	AnswerID         int64     `json:"answer_id"`
	QuestionID       int64     `json:"question_id"`
	QuestionText     string    `json:"question_text"`
	Options          []string  `json:"options"`
	SelectedAnswers  []string  `json:"selected_answers"`
	CorrectAnswers   []string  `json:"correct_answers"`
	Correct          bool      `json:"correct"`
	TimeTakenSeconds float64   `json:"time_taken_seconds"`
	TimeScore        float64   `json:"time_score"`
	PerformanceScore float64   `json:"performance_score"`
	EloChange        int       `json:"elo_change"`
	AnsweredAt       time.Time `json:"answered_at"`
}

type SessionHistoryResponse struct {
	Session QuizSession            `json:"session"`
	Answers []SessionAnswerHistory `json:"answers"`
}
