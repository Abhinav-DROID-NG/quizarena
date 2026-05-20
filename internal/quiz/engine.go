package quiz

import "math"

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type RankTier string

const (
	RankBronze   RankTier = "Bronze"
	RankSilver   RankTier = "Silver"
	RankGold     RankTier = "Gold"
	RankPlatinum RankTier = "Platinum"
	RankDiamond  RankTier = "Diamond"
	RankMaster   RankTier = "Master"
)

func TimeScore(timeTakenSeconds, expectedTimeSeconds float64) float64 {
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

func PerformanceScore(timeTakenSeconds, expectedTimeSeconds float64, correct bool) float64 {
	correctness := 0.0
	if correct {
		correctness = 1
	}
	return (0.8 * TimeScore(timeTakenSeconds, expectedTimeSeconds)) + (0.2 * correctness)
}

func ExpectedScore(userElo, questionElo float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, (questionElo-userElo)/400.0))
}

func KFactor(d Difficulty) float64 {
	switch d {
	case DifficultyEasy:
		return 16
	case DifficultyHard:
		return 32
	default:
		return 24
	}
}

func ApplyPenalties(basePerformance, timeTakenSeconds, expectedTimeSeconds float64, correct, skipped bool, consecutiveSkips, guessStreak int) float64 {
	performance := basePerformance

	if !correct && expectedTimeSeconds > 0 && timeTakenSeconds < (expectedTimeSeconds*0.25) {
		performance -= 0.08
	}

	if skipped && consecutiveSkips > 0 {
		skipPenalty := math.Min(0.15, float64(consecutiveSkips)*0.05)
		performance -= skipPenalty
	}

	if !correct && guessStreak > 0 {
		guessPenalty := math.Min(0.12, float64(guessStreak)*0.04)
		performance -= guessPenalty
	}

	if performance < 0 {
		return 0
	}
	if performance > 1 {
		return 1
	}
	return performance
}

func UpdateElo(userElo, questionElo float64, difficulty Difficulty, actualScore float64) (newElo float64, eloChange int) {
	expected := ExpectedScore(userElo, questionElo)
	change := KFactor(difficulty) * (actualScore - expected)
	newElo = userElo + change
	return newElo, int(math.Round(change))
}

func RankForElo(elo float64) RankTier {
	switch {
	case elo >= 2400:
		return RankMaster
	case elo >= 2000:
		return RankDiamond
	case elo >= 1600:
		return RankPlatinum
	case elo >= 1200:
		return RankGold
	case elo >= 800:
		return RankSilver
	default:
		return RankBronze
	}
}
