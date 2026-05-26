package utils

import (
	"testing"

	"github.com/Abhinav-DROID-NG/quizarena/models"
)

func TestValidateSubject(t *testing.T) {
	if err := ValidateSubject("Computer Networks"); err != nil {
		t.Fatalf("expected valid subject, got %v", err)
	}
	if err := ValidateSubject("bad;drop table"); err == nil {
		t.Fatal("expected invalid subject")
	}
}

func TestValidateQuestionInput(t *testing.T) {
	question := models.Question{
		Subject:             "Algorithms",
		Type:                "MCQ",
		Difficulty:          models.DifficultyMedium,
		QuestionText:        "What is the complexity of binary search?",
		Options:             []string{"O(log n)", "O(n)"},
		CorrectAnswers:      []string{"O(log n)"},
		QuestionElo:         1300,
		ExpectedTimeSeconds: 30,
	}
	if err := ValidateQuestionInput(question); err != nil {
		t.Fatalf("expected valid question: %v", err)
	}
	question.CorrectAnswers = []string{"O(n^2)"}
	if err := ValidateQuestionInput(question); err == nil {
		t.Fatal("expected invalid question")
	}
}

func TestValidateQuestionInputRejectsControlChars(t *testing.T) {
	question := models.Question{
		Subject:             "Algorithms",
		Type:                "MCQ",
		Difficulty:          models.DifficultyMedium,
		QuestionText:        "Bad\x01Text",
		Options:             []string{"A", "B"},
		CorrectAnswers:      []string{"A"},
		QuestionElo:         1200,
		ExpectedTimeSeconds: 30,
	}
	if err := ValidateQuestionInput(question); err == nil {
		t.Fatal("expected invalid question with control chars")
	}
}

func TestValidateSelectedAnswersRejectsDuplicates(t *testing.T) {
	if err := ValidateSelectedAnswers([]string{"A", "A"}); err == nil {
		t.Fatal("expected invalid selected answers with duplicates")
	}
}

func TestClampElo(t *testing.T) {
	if got := ClampElo(-10); got != MinUserElo {
		t.Fatalf("expected min elo, got %d", got)
	}
	if got := ClampElo(9000); got != MaxUserElo {
		t.Fatalf("expected max elo, got %d", got)
	}
}
