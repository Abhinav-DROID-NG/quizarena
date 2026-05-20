package models

type User struct {
	ID                  int64   `json:"id"`
	GoogleSub           string  `json:"-"`
	Email               string  `json:"email"`
	Name                string  `json:"name"`
	Picture             string  `json:"picture"`
	CurrentElo          int     `json:"current_elo"`
	PeakElo             int     `json:"peak_elo"`
	AccuracyPercentage  float64 `json:"accuracy_percentage"`
	AverageResponseTime float64 `json:"average_response_time"`
	TotalQuestions      int     `json:"total_questions_solved"`
	StrongestSubject    string  `json:"strongest_subject"`
	WeakestSubject      string  `json:"weakest_subject"`
}
