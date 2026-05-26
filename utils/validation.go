package utils

import (
	"errors"
	"strings"

	"github.com/Abhinav-DROID-NG/quizarena/models"
)

const (
	MinPort               = 1
	MaxPort               = 65535
	MinDBMaxConns         = int32(1)
	MaxDBMaxConns         = int32(100)
	MinShutdownTimeoutSec = 1
	MaxShutdownTimeoutSec = 120
	MinJWTExpirationHours = 1
	MaxJWTExpirationHours = 24 * 30

	MinUserElo = 0
	MaxUserElo = 5000

	MinExpectedTimeSeconds = 5.0
	MaxExpectedTimeSeconds = 600.0
	MaxAnswerTimeSeconds   = 3600.0

	MaxNameLength         = 255
	MaxEmailLength        = 320
	MaxSubjectLength      = 100
	MaxQuestionTextLength = 5000
	MaxOptionLength       = 512
	MinQuestionOptions    = 2
	MaxQuestionOptions    = 10
)

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidName        = errors.New("invalid name")
	ErrInvalidSubject     = errors.New("invalid subject")
	ErrInvalidElo         = errors.New("invalid elo")
	ErrInvalidTimeRange   = errors.New("invalid time range")
	ErrInvalidDifficulty  = errors.New("invalid difficulty")
	ErrInvalidQuestion    = errors.New("invalid question")
	ErrInvalidAnswerSet   = errors.New("invalid selected answers")
	ErrInvalidConfigValue = errors.New("invalid configuration value")
)

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > MaxEmailLength {
		return ErrInvalidEmail
	}
	at := strings.IndexByte(email, '@')
	dot := strings.LastIndexByte(email, '.')
	if at <= 0 || dot <= at+1 || dot == len(email)-1 {
		return ErrInvalidEmail
	}
	return nil
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > MaxNameLength {
		return ErrInvalidName
	}
	return nil
}

func ValidateSubject(subject string) error {
	subject = strings.TrimSpace(subject)
	if len(subject) > MaxSubjectLength {
		return ErrInvalidSubject
	}
	for _, ch := range subject {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			continue
		}
		switch ch {
		case ' ', '-', '_', '&', '/':
			continue
		default:
			return ErrInvalidSubject
		}
	}
	return nil
}

func ValidateElo(elo int) error {
	if elo < MinUserElo || elo > MaxUserElo {
		return ErrInvalidElo
	}
	return nil
}

func ClampElo(elo int) int {
	if elo < MinUserElo {
		return MinUserElo
	}
	if elo > MaxUserElo {
		return MaxUserElo
	}
	return elo
}

func ValidateTimeTaken(seconds float64) error {
	if seconds <= 0 || seconds > MaxAnswerTimeSeconds {
		return ErrInvalidTimeRange
	}
	return nil
}

func ValidateExpectedTime(seconds float64) error {
	if seconds < MinExpectedTimeSeconds || seconds > MaxExpectedTimeSeconds {
		return ErrInvalidTimeRange
	}
	return nil
}

func ValidateDifficulty(d models.Difficulty) error {
	switch d {
	case models.DifficultyEasy, models.DifficultyMedium, models.DifficultyHard:
		return nil
	default:
		return ErrInvalidDifficulty
	}
}

func ValidateQuestionInput(q models.Question) error {
	if ValidateSubject(q.Subject) != nil {
		return ErrInvalidQuestion
	}
	if ValidateDifficulty(q.Difficulty) != nil {
		return ErrInvalidQuestion
	}
	if q.Type != "MCQ" && q.Type != "MSQ" {
		return ErrInvalidQuestion
	}
	text := strings.TrimSpace(q.QuestionText)
	if text == "" || len(text) > MaxQuestionTextLength {
		return ErrInvalidQuestion
	}
	if hasControlChars(text) {
		return ErrInvalidQuestion
	}
	if len(q.Options) < MinQuestionOptions || len(q.Options) > MaxQuestionOptions {
		return ErrInvalidQuestion
	}
	options := make(map[string]struct{}, len(q.Options))
	for _, option := range q.Options {
		trimmed := strings.TrimSpace(option)
		if trimmed == "" || len(trimmed) > MaxOptionLength || hasControlChars(trimmed) {
			return ErrInvalidQuestion
		}
		if _, exists := options[trimmed]; exists {
			return ErrInvalidQuestion
		}
		options[trimmed] = struct{}{}
	}
	if len(q.CorrectAnswers) == 0 {
		return ErrInvalidQuestion
	}
	correctAnswers := make(map[string]struct{}, len(q.CorrectAnswers))
	for _, ans := range q.CorrectAnswers {
		trimmed := strings.TrimSpace(ans)
		if _, ok := options[trimmed]; !ok {
			return ErrInvalidQuestion
		}
		if _, exists := correctAnswers[trimmed]; exists {
			return ErrInvalidQuestion
		}
		correctAnswers[trimmed] = struct{}{}
	}
	if q.Type == "MCQ" && len(correctAnswers) != 1 {
		return ErrInvalidQuestion
	}
	if ValidateElo(q.QuestionElo) != nil {
		return ErrInvalidQuestion
	}
	if ValidateExpectedTime(q.ExpectedTimeSeconds) != nil {
		return ErrInvalidQuestion
	}
	return nil
}

func ValidateSelectedAnswers(selected []string) error {
	if len(selected) == 0 {
		return ErrInvalidAnswerSet
	}
	seen := make(map[string]struct{}, len(selected))
	for _, answer := range selected {
		trimmed := strings.TrimSpace(answer)
		if trimmed == "" || hasControlChars(trimmed) {
			return ErrInvalidAnswerSet
		}
		if _, exists := seen[trimmed]; exists {
			return ErrInvalidAnswerSet
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}

func hasControlChars(input string) bool {
	for _, r := range input {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
}
