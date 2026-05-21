package services

import (
	"math"
	"math/rand"

	"github.com/Abhinav-DROID-NG/quizarena/models"
)

type EloEngine struct{}

func NewEloEngine() *EloEngine { return &EloEngine{} }

func (e *EloEngine) TimeScore(timeTakenSeconds, expectedTimeSeconds float64) float64 {
	if expectedTimeSeconds <= 0 {
		return 0
	}
	score := 1 - (timeTakenSeconds / expectedTimeSeconds)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func (e *EloEngine) PerformanceScore(timeScore float64, correct bool) float64 {
	if !correct {
		return 0
	}
	// For correct answers, base performance is 0.5 (average) + up to 0.5 for speed
	return 0.5 + (0.5 * timeScore)
}

func (e *EloEngine) ExpectedProbability(userElo, questionElo int) float64 {
	return 1 / (1 + math.Pow(10, float64(questionElo-userElo)/400))
}

func (e *EloEngine) KFactor(d models.Difficulty) float64 {
	switch d {
	case models.DifficultyEasy:
		return 16
	case models.DifficultyHard:
		return 32
	default:
		return 24
	}
}

func (e *EloEngine) CalculateNewElo(userElo, questionElo int, d models.Difficulty, performance float64) (newElo int, delta int) {
	expected := e.ExpectedProbability(userElo, questionElo)
	k := e.KFactor(d)
	change := int(math.Round(k * (performance - expected)))

	// If performance is 0 (wrong answer), ensure delta is at least -1
	if performance == 0 && change >= 0 {
		change = -1
	}

	return userElo + change, change
}

func (e *EloEngine) ApplyAntiGuessingPenalty(delta int, correct bool, timeTaken, expectedTime float64, consecutiveSkips int) int {
	if !correct && expectedTime > 0 && timeTaken <= expectedTime*0.15 {
		delta -= 4
	}
	if consecutiveSkips >= 2 {
		delta -= consecutiveSkips
	}
	return delta
}

func (e *EloEngine) NextQuestionDifficulty(userElo, nextTargetElo int) models.Difficulty {
	diff := nextTargetElo - userElo
	switch {
	case diff >= 50:
		return models.DifficultyHard
	case diff <= -50:
		return models.DifficultyEasy
	default:
		return models.DifficultyMedium
	}
}

func (e *EloEngine) NextTargetElo(userElo int) int {
	roll := rand.Float64()
	delta := rand.Intn(51) + 25
	switch {
	case roll < 0.7:
		if rand.Intn(2) == 0 {
			return userElo + delta/2
		}
		return userElo - delta/2
	case roll < 0.9:
		return userElo + delta
	default:
		return userElo - delta
	}
}

func (e *EloEngine) ConfidenceScore(correct bool, timeScore float64, consecutiveWrong, consecutiveSkips int) float64 {
	confidence := 0.6 + (timeScore * 0.4)
	if !correct {
		confidence -= 0.15
	}
	confidence -= float64(consecutiveWrong) * 0.05
	confidence -= float64(consecutiveSkips) * 0.03
	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}
