package models

type QuizSession struct {
	ID      int64  `json:"id"`
	UserID  int64  `json:"user_id"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
}

type AnswerResponse struct {
	Correct                bool    `json:"correct"`
	CorrectAnswer          string  `json:"correct_answer"`
	TimeTaken              float64 `json:"time_taken"`
	TimeScore              float64 `json:"time_score"`
	PerformanceScore       float64 `json:"performance_score"`
	EloChange              int     `json:"elo_change"`
	NewUserElo             int     `json:"new_user_elo"`
	NextQuestionDifficulty string  `json:"next_question_difficulty"`
	ConfidenceScore        float64 `json:"confidence_score"`
}
