package models

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type Question struct {
	ID                  int64      `json:"id"`
	Subject             string     `json:"subject"`
	Difficulty          Difficulty `json:"difficulty"`
	QuestionText        string     `json:"question_text"`
	Options             []string   `json:"options"`
	CorrectAnswer       string     `json:"-"`
	QuestionElo         int        `json:"question_elo"`
	ExpectedTimeSeconds float64    `json:"expected_time_seconds"`
}
