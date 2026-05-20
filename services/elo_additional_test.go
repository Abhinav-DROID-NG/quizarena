package services

import (
	"testing"

	"github.com/Abhinav-DROID-NG/quizarena/models"
)

func TestApplyAntiGuessingPenalty(t *testing.T) {
	engine := NewEloEngine()
	base := 10
	got := engine.ApplyAntiGuessingPenalty(base, false, 1, 20, 3)
	want := 3 // 10 - 4 (fast wrong) - 3 (skip streak)
	if got != want {
		t.Fatalf("expected %d got %d", want, got)
	}
}

func TestConfidenceScoreClampedRange(t *testing.T) {
	engine := NewEloEngine()
	if got := engine.ConfidenceScore(true, 10, 0, 0); got != 1 {
		t.Fatalf("expected upper clamp 1 got %v", got)
	}
	if got := engine.ConfidenceScore(false, 0, 20, 20); got != 0 {
		t.Fatalf("expected lower clamp 0 got %v", got)
	}
}

func TestNextQuestionDifficulty(t *testing.T) {
	engine := NewEloEngine()
	if got := engine.NextQuestionDifficulty(1200, 1260); got != models.DifficultyHard {
		t.Fatalf("expected hard got %s", got)
	}
	if got := engine.NextQuestionDifficulty(1200, 1140); got != models.DifficultyEasy {
		t.Fatalf("expected easy got %s", got)
	}
	if got := engine.NextQuestionDifficulty(1200, 1230); got != models.DifficultyMedium {
		t.Fatalf("expected medium got %s", got)
	}
}
