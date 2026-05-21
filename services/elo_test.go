package services

import (
	"math"
	"testing"

	"github.com/Abhinav-DROID-NG/quizarena/models"
)

func TestExpectedProbability(t *testing.T) {
	engine := NewEloEngine()
	got := engine.ExpectedProbability(1400, 1500)
	want := 1 / (1 + math.Pow(10, (1500.0-1400.0)/400.0))
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("expected %.10f got %.10f", want, got)
	}
}

func TestPerformanceScore(t *testing.T) {
	engine := NewEloEngine()
	timeScore := engine.TimeScore(10, 20)
	if timeScore != 0.5 {
		t.Fatalf("expected time score 0.5 got %v", timeScore)
	}
	perfCorrect := engine.PerformanceScore(timeScore, true)
	if math.Abs(perfCorrect-0.75) > 1e-9 {
		t.Fatalf("expected performance 0.75 got %v", perfCorrect)
	}
	perfWrong := engine.PerformanceScore(timeScore, false)
	if perfWrong != 0 {
		t.Fatalf("expected performance 0 for wrong answer got %v", perfWrong)
	}
}

func TestKFactorByDifficulty(t *testing.T) {
	engine := NewEloEngine()
	if engine.KFactor(models.DifficultyEasy) != 16 {
		t.Fatal("easy k-factor mismatch")
	}
	if engine.KFactor(models.DifficultyMedium) != 24 {
		t.Fatal("medium k-factor mismatch")
	}
	if engine.KFactor(models.DifficultyHard) != 32 {
		t.Fatal("hard k-factor mismatch")
	}
}
