package quiz

import (
	"math"
	"testing"
)

func TestTimeScore(t *testing.T) {
	if got := TimeScore(5, 10); got != 0.5 {
		t.Fatalf("expected 0.5 got %v", got)
	}
	if got := TimeScore(20, 10); got != 0 {
		t.Fatalf("expected floor at 0 got %v", got)
	}
}

func TestPerformanceScore(t *testing.T) {
	got := PerformanceScore(5, 10, true)
	want := 0.8*0.5 + 0.2
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("expected %v got %v", want, got)
	}
}

func TestApplyPenalties(t *testing.T) {
	base := 0.6
	got := ApplyPenalties(base, 1, 10, false, true, 2, 2)
	if !(got < base) {
		t.Fatalf("expected penalized performance lower than base; base=%v got=%v", base, got)
	}
}

func TestKFactor(t *testing.T) {
	if k := KFactor(DifficultyEasy); k != 16 {
		t.Fatalf("expected 16 got %v", k)
	}
	if k := KFactor(DifficultyMedium); k != 24 {
		t.Fatalf("expected 24 got %v", k)
	}
	if k := KFactor(DifficultyHard); k != 32 {
		t.Fatalf("expected 32 got %v", k)
	}
}

func TestRankForElo(t *testing.T) {
	cases := []struct {
		elo  float64
		rank RankTier
	}{
		{750, RankBronze},
		{900, RankSilver},
		{1300, RankGold},
		{1750, RankPlatinum},
		{2100, RankDiamond},
		{2500, RankMaster},
	}
	for _, tc := range cases {
		if got := RankForElo(tc.elo); got != tc.rank {
			t.Fatalf("elo=%v expected %s got %s", tc.elo, tc.rank, got)
		}
	}
}
