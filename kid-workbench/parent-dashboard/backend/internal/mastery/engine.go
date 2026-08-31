package mastery

import (
	"math"
	"time"

	"github.com/conchi/study-workbench/internal/config"
)

type Status string

const (
	StatusNotStarted Status = "not_started"
	StatusLearning   Status = "learning"
	StatusShaky      Status = "shaky"
	StatusMastered   Status = "mastered"
	StatusReviewDue  Status = "review_due" // 派生状态，只用于展示
)

const (
	SourceQuiz       = "quiz"
	SourceGame       = "game"
	SourceParentMark = "parent_mark"
)

type Config = config.Mastery

func DefaultConfig() Config {
	return Config{
		BaseMasterStreak: 2,
		MinAccuracy:      0.8,
		ShakyMinAttempts: 3,
		ShakyAccuracy:    0.6,
		EaseMin:          1.3,
		EaseMax:          2.8,
		EaseUp:           0.1,
		EaseDown:         0.2,
		MaxIntervalDays:  60,
	}
}

type State struct {
	Status       Status
	Attempts     int
	Correct      int
	Streak       int
	BestStreak   int
	Ease         float64
	IntervalDays int
	DueAt        time.Time
	FirstSeenAt  time.Time
	MasteredAt   *time.Time
}

func NewState() State {
	return State{Status: StatusNotStarted, Ease: 2.5}
}

func (s State) Accuracy() float64 {
	if s.Attempts == 0 {
		return 0
	}
	return float64(s.Correct) / float64(s.Attempts)
}

type Attempt struct {
	Correct bool
	At      time.Time
	Source  string
}

// Apply 计算一次作答后的新状态。纯函数，不修改入参。
func Apply(s State, a Attempt, difficulty int, cfg Config) State {
	if s.FirstSeenAt.IsZero() {
		s.FirstSeenAt = a.At
	}
	if a.Source == SourceParentMark {
		return markMastered(s, a.At, difficulty, cfg)
	}

	s.Attempts++
	if a.Correct {
		s.Correct++
	}

	need := masterStreak(difficulty, cfg)

	if a.Correct {
		s.Streak++
		if s.Streak > s.BestStreak {
			s.BestStreak = s.Streak
		}
		s.Ease = clamp(s.Ease+cfg.EaseUp, cfg.EaseMin, cfg.EaseMax)

		if s.Streak >= need && s.Accuracy() >= cfg.MinAccuracy {
			s.Status = StatusMastered
			prev := s.IntervalDays
			if s.MasteredAt == nil {
				prev = 0 // 首次掌握从 1 天开始：1 → 3 → 7 → …
			}
			s.IntervalDays = nextInterval(prev, s.Ease, cfg.MaxIntervalDays)
			s.DueAt = a.At.AddDate(0, 0, s.IntervalDays)
			if s.MasteredAt == nil {
				at := a.At
				s.MasteredAt = &at
			}
			return s
		}

		s.Status = StatusLearning
		s.IntervalDays = 1
		s.DueAt = a.At.AddDate(0, 0, 1)
		return s
	}

	s.Streak = 0
	s.Ease = clamp(s.Ease-cfg.EaseDown, cfg.EaseMin, cfg.EaseMax)
	s.IntervalDays = 1
	s.DueAt = a.At.AddDate(0, 0, 1)
	if s.Attempts >= cfg.ShakyMinAttempts && s.Accuracy() < cfg.ShakyAccuracy {
		s.Status = StatusShaky
	} else {
		s.Status = StatusLearning
	}
	return s
}

// Replay 从完整作答历史重建状态，用于撤销家长标记或重算聚合表
func Replay(history []Attempt, difficulty int, cfg Config) State {
	s := NewState()
	for _, a := range history {
		s = Apply(s, a, difficulty, cfg)
	}
	return s
}

// Display 把 mastered + 过期 折算成 review_due
func Display(s State, now time.Time) Status {
	if s.Status == StatusMastered && !s.DueAt.IsZero() && s.DueAt.Before(now) {
		return StatusReviewDue
	}
	return s.Status
}

func markMastered(s State, at time.Time, difficulty int, cfg Config) State {
	if need := masterStreak(difficulty, cfg); s.Streak < need {
		s.Streak = need
	}
	if s.Streak > s.BestStreak {
		s.BestStreak = s.Streak
	}
	s.Status = StatusMastered
	s.IntervalDays = 3
	s.DueAt = at.AddDate(0, 0, 3)
	if s.MasteredAt == nil {
		t := at
		s.MasteredAt = &t
	}
	return s
}

func masterStreak(difficulty int, cfg Config) int {
	if difficulty < 1 {
		difficulty = 1
	}
	return cfg.BaseMasterStreak + difficulty
}

func nextInterval(cur int, ease float64, max int) int {
	if cur <= 0 {
		return 1
	}
	n := int(math.Round(float64(cur) * ease))
	if n <= cur {
		n = cur + 1
	}
	if n > max {
		n = max
	}
	return n
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}
