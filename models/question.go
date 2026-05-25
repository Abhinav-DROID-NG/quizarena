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
	Type                string     `json:"type"` // MCQ, MSQ
	Difficulty          Difficulty `json:"difficulty"`
	QuestionText        string     `json:"question_text"`
	Options             []string   `json:"options"`
	CorrectAnswers      []string   `json:"-"`
	QuestionElo         int        `json:"question_elo"`
	ExpectedTimeSeconds float64    `json:"expected_time_seconds"`
}
