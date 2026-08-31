package mastery_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-workbench/internal/mastery"
)

var base = time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
var cfg = mastery.DefaultConfig()

func correctAt(d int) mastery.Attempt {
	return mastery.Attempt{Correct: true, At: base.AddDate(0, 0, d), Source: mastery.SourceQuiz}
}

func wrongAt(d int) mastery.Attempt {
	return mastery.Attempt{Correct: false, At: base.AddDate(0, 0, d), Source: mastery.SourceQuiz}
}

func TestThreeCorrectInARowBecomesMastered(t *testing.T) {
	s := mastery.NewState()
	for i := 0; i < 3; i++ {
		s = mastery.Apply(s, correctAt(i), 1, cfg)
	}
	require.Equal(t, mastery.StatusMastered, s.Status)
	require.Equal(t, 3, s.Streak)
	require.Equal(t, 1, s.IntervalDays)
	require.NotNil(t, s.MasteredAt)
	require.Equal(t, base.AddDate(0, 0, 3), s.DueAt)
}

func TestHarderKpNeedsMoreStreak(t *testing.T) {
	s := mastery.NewState()
	for i := 0; i < 4; i++ {
		s = mastery.Apply(s, correctAt(i), 3, cfg)
	}
	require.Equal(t, mastery.StatusLearning, s.Status)
	s = mastery.Apply(s, correctAt(4), 3, cfg)
	require.Equal(t, mastery.StatusMastered, s.Status)
}

func TestIntervalGrowsOnRepeatedSuccess(t *testing.T) {
	s := mastery.NewState()
	for i := 0; i < 3; i++ {
		s = mastery.Apply(s, correctAt(i), 1, cfg)
	}
	first := s.IntervalDays
	s = mastery.Apply(s, correctAt(4), 1, cfg)
	require.Greater(t, s.IntervalDays, first)
	require.Equal(t, base.AddDate(0, 0, 4+s.IntervalDays), s.DueAt)
}

func TestMasteredThenWrongFallsBack(t *testing.T) {
	s := mastery.NewState()
	for i := 0; i < 3; i++ {
		s = mastery.Apply(s, correctAt(i), 1, cfg)
	}
	s = mastery.Apply(s, wrongAt(4), 1, cfg)
	require.Equal(t, mastery.StatusLearning, s.Status)
	require.Zero(t, s.Streak)
	require.Equal(t, 1, s.IntervalDays)
	require.Equal(t, 3, s.BestStreak)
}

func TestThreeWrongOutOfFiveBecomesShaky(t *testing.T) {
	s := mastery.NewState()
	s = mastery.Apply(s, wrongAt(0), 1, cfg)
	s = mastery.Apply(s, correctAt(1), 1, cfg)
	s = mastery.Apply(s, wrongAt(2), 1, cfg)
	s = mastery.Apply(s, correctAt(3), 1, cfg)
	s = mastery.Apply(s, wrongAt(4), 1, cfg)
	require.Equal(t, mastery.StatusShaky, s.Status)
	require.InDelta(t, 0.4, s.Accuracy(), 0.001)
}

func TestEaseStaysInBounds(t *testing.T) {
	s := mastery.NewState()
	for i := 0; i < 20; i++ {
		s = mastery.Apply(s, correctAt(i), 1, cfg)
	}
	require.LessOrEqual(t, s.Ease, cfg.EaseMax)

	s2 := mastery.NewState()
	for i := 0; i < 20; i++ {
		s2 = mastery.Apply(s2, wrongAt(i), 1, cfg)
	}
	require.GreaterOrEqual(t, s2.Ease, cfg.EaseMin)
	require.LessOrEqual(t, s.IntervalDays, cfg.MaxIntervalDays)
}

func TestParentMarkDoesNotCountAsPractice(t *testing.T) {
	s := mastery.NewState()
	s = mastery.Apply(s, mastery.Attempt{Correct: true, At: base, Source: mastery.SourceParentMark}, 1, cfg)
	require.Equal(t, mastery.StatusMastered, s.Status)
	require.Zero(t, s.Attempts)
	require.Equal(t, 3, s.IntervalDays)
}

func TestDisplayStatusTurnsReviewDueWhenOverdue(t *testing.T) {
	s := mastery.NewState()
	for i := 0; i < 3; i++ {
		s = mastery.Apply(s, correctAt(i), 1, cfg)
	}
	require.Equal(t, mastery.StatusMastered, mastery.Display(s, s.DueAt.Add(-time.Hour)))
	require.Equal(t, mastery.StatusReviewDue, mastery.Display(s, s.DueAt.Add(time.Hour)))
}

func TestReplayReproducesState(t *testing.T) {
	history := []mastery.Attempt{correctAt(0), wrongAt(1), correctAt(2), correctAt(3), correctAt(4)}

	step := mastery.NewState()
	for _, a := range history {
		step = mastery.Apply(step, a, 2, cfg)
	}
	require.Equal(t, step, mastery.Replay(history, 2, cfg))
}
