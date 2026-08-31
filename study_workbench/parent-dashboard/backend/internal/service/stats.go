package service

import (
	"time"

	"github.com/conchi/study-workbench/internal/repo"
)

type StatsService struct{ repo *repo.Repo }

func NewStatsService(r *repo.Repo) *StatsService { return &StatsService{repo: r} }

type TrendPoint struct {
	Date               string `json:"date"`
	PracticeMin        int    `json:"practice_min"`
	Attempts           int    `json:"attempts"`
	Correct            int    `json:"correct"`
	NewlyMastered      int    `json:"newly_mastered"`
	CumulativeMastered int    `json:"cumulative_mastered"`
}

func (s *StatsService) Trend(childID int64, days int) ([]TrendPoint, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	start := time.Now().AddDate(0, 0, -(days - 1))

	type row struct {
		StatDate      string
		PracticeSec   int
		Attempts      int
		Correct       int
		NewlyMastered int
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT stat_date, practice_sec, attempts, correct, newly_mastered
		FROM daily_stats WHERE child_id = ? AND stat_date >= ?
		ORDER BY stat_date`, childID, start.Format("2006-01-02")).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byDate := map[string]row{}
	for _, r := range rows {
		key := r.StatDate
		if len(key) > 10 {
			key = key[:10]
		}
		byDate[key] = r
	}

	var baseline int
	_ = s.repo.DB().Raw(`
		SELECT COALESCE(SUM(newly_mastered),0) FROM daily_stats
		WHERE child_id = ? AND stat_date < ?`, childID, start.Format("2006-01-02")).
		Scan(&baseline).Error

	out := make([]TrendPoint, 0, days)
	cum := baseline
	for i := 0; i < days; i++ {
		key := start.AddDate(0, 0, i).Format("2006-01-02")
		r := byDate[key]
		cum += r.NewlyMastered
		out = append(out, TrendPoint{
			Date: key, PracticeMin: r.PracticeSec / 60, Attempts: r.Attempts,
			Correct: r.Correct, NewlyMastered: r.NewlyMastered, CumulativeMastered: cum,
		})
	}
	return out, nil
}

type CalendarDay struct {
	Date        string `json:"date"`
	PracticeMin int    `json:"practice_min"`
	Attempts    int    `json:"attempts"`
	Mastered    int    `json:"mastered"`
	CheckedIn   bool   `json:"checked_in"`
}

func (s *StatsService) Calendar(childID int64, months int) ([]CalendarDay, error) {
	if months <= 0 || months > 24 {
		months = 3
	}
	start := time.Now().AddDate(0, -months, 0).Format("2006-01-02")

	type row struct {
		StatDate      string
		PracticeSec   int
		Attempts      int
		NewlyMastered int
		CheckedIn     bool
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT stat_date, practice_sec, attempts, newly_mastered, checked_in
		FROM daily_stats
		WHERE child_id = ? AND stat_date >= ? AND attempts > 0
		ORDER BY stat_date`, childID, start).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]CalendarDay, 0, len(rows))
	for _, r := range rows {
		date := r.StatDate
		if len(date) > 10 {
			date = date[:10]
		}
		out = append(out, CalendarDay{
			Date: date, PracticeMin: r.PracticeSec / 60, Attempts: r.Attempts,
			Mastered: r.NewlyMastered, CheckedIn: r.CheckedIn,
		})
	}
	return out, nil
}

type ReviewItem struct {
	KpID        int64   `json:"kp_id"`
	Title       string  `json:"title"`
	SubjectCode string  `json:"subject_code"`
	SubjectName string  `json:"subject_name"`
	Status      string  `json:"status"`
	Accuracy    float64 `json:"accuracy"`
}

func (s *StatsService) ReviewQueue(childID int64, limit int) ([]ReviewItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var out []ReviewItem
	err := s.repo.DB().Raw(`
		SELECT kp.id AS kp_id, kp.title, s.code AS subject_code, s.name AS subject_name,
		       `+effectiveStatusSQL+` AS status,
		       CASE WHEN COALESCE(ms.attempts,0) > 0
		            THEN CAST(ms.correct AS REAL) / ms.attempts ELSE 0 END AS accuracy
		FROM knowledge_points kp
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		ORDER BY CASE `+effectiveStatusSQL+`
		           WHEN 'shaky' THEN 0 WHEN 'review_due' THEN 1
		           WHEN 'learning' THEN 2 WHEN 'not_started' THEN 3 ELSE 4 END,
		         accuracy ASC, ms.due_at ASC, kp.order_no
		LIMIT ?`, childID, limit).Scan(&out).Error
	return out, err
}
