package service

import (
	"fmt"
	"time"

	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
)

type DashboardService struct{ repo *repo.Repo }

func NewDashboardService(r *repo.Repo) *DashboardService {
	return &DashboardService{repo: r}
}

type StatusCounts struct {
	NotStarted int `json:"not_started"`
	Learning   int `json:"learning"`
	Shaky      int `json:"shaky"`
	Mastered   int `json:"mastered"`
	ReviewDue  int `json:"review_due"`
}

type ChildDTO struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Grade   string `json:"grade"`
	Flowers int    `json:"flowers"`
}

type Overview struct {
	Child      ChildDTO     `json:"child"`
	TotalKp    int          `json:"total_kp"`
	Counts     StatusCounts `json:"counts"`
	WeekDelta  WeekDelta    `json:"week_delta"`
	StreakDays int          `json:"streak_days"`
	Today      TodayStat    `json:"today"`
}

type WeekDelta struct {
	Mastered    int `json:"mastered"`
	PracticeMin int `json:"practice_min"`
}

type TodayStat struct {
	PracticeMin int `json:"practice_min"`
	Attempts    int `json:"attempts"`
	TasksDone   int `json:"tasks_done"`
	TasksTotal  int `json:"tasks_total"`
}

const effectiveStatusSQL = `
	CASE WHEN ms.status = 'mastered' AND ms.due_at IS NOT NULL AND ms.due_at < CURRENT_TIMESTAMP
	     THEN 'review_due' ELSE COALESCE(ms.status, 'not_started') END`

func (s *DashboardService) Overview(childID int64) (Overview, error) {
	var out Overview

	var child model.Child
	if err := s.repo.DB().First(&child, childID).Error; err != nil {
		return out, err
	}
	out.Child = ChildDTO{ID: child.ID, Name: child.Name, Grade: child.Grade, Flowers: child.Flowers}

	var total int64
	if err := s.repo.DB().Model(&model.KnowledgePoint{}).Count(&total).Error; err != nil {
		return out, err
	}
	out.TotalKp = int(total)

	type row struct {
		Status string
		N      int
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT `+effectiveStatusSQL+` AS status, COUNT(1) AS n
		FROM mastery_states ms WHERE ms.child_id = ?
		GROUP BY 1`, childID).Scan(&rows).Error; err != nil {
		return out, err
	}

	tracked := 0
	for _, r := range rows {
		tracked += r.N
		switch r.Status {
		case "learning":
			out.Counts.Learning = r.N
		case "shaky":
			out.Counts.Shaky = r.N
		case "mastered":
			out.Counts.Mastered = r.N
		case "review_due":
			out.Counts.ReviewDue = r.N
		}
	}
	out.Counts.NotStarted = out.TotalKp - tracked

	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	var wd struct {
		Mastered    int
		PracticeSec int
	}
	if err := s.repo.DB().Raw(`
		SELECT COALESCE(SUM(newly_mastered),0) AS mastered,
		       COALESCE(SUM(practice_sec),0)   AS practice_sec
		FROM daily_stats WHERE child_id = ? AND stat_date >= ?`, childID, weekAgo).
		Scan(&wd).Error; err != nil {
		return out, err
	}
	out.WeekDelta = WeekDelta{Mastered: wd.Mastered, PracticeMin: wd.PracticeSec / 60}

	today := time.Now().Format("2006-01-02")
	var td struct {
		PracticeSec int
		Attempts    int
	}
	_ = s.repo.DB().Raw(`
		SELECT COALESCE(practice_sec,0) AS practice_sec, COALESCE(attempts,0) AS attempts
		FROM daily_stats WHERE child_id = ? AND stat_date = ?`, childID, today).
		Scan(&td).Error
	out.Today = TodayStat{PracticeMin: td.PracticeSec / 60, Attempts: td.Attempts}

	var taskAgg struct {
		Done  int
		Total int
	}
	_ = s.repo.DB().Raw(`
		SELECT COALESCE(SUM(CASE WHEN status='done' THEN 1 ELSE 0 END),0) AS done,
		       COUNT(1) AS total
		FROM study_plans WHERE child_id = ? AND plan_date = ?`, childID, today).
		Scan(&taskAgg).Error
	out.Today.TasksDone, out.Today.TasksTotal = taskAgg.Done, taskAgg.Total

	streak, err := s.streakDays(childID)
	if err != nil {
		return out, err
	}
	out.StreakDays = streak
	return out, nil
}

func (s *DashboardService) streakDays(childID int64) (int, error) {
	var dates []string
	if err := s.repo.DB().Raw(`
		SELECT stat_date FROM daily_stats
		WHERE child_id = ? AND attempts > 0
		ORDER BY stat_date DESC LIMIT 400`, childID).Scan(&dates).Error; err != nil {
		return 0, err
	}
	set := map[string]bool{}
	for _, d := range dates {
		if len(d) >= 10 {
			set[d[:10]] = true
		}
	}

	streak := 0
	day := time.Now()
	if !set[day.Format("2006-01-02")] {
		day = day.AddDate(0, 0, -1)
	}
	for set[day.Format("2006-01-02")] {
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak, nil
}

type SubjectSummary struct {
	Code     string       `json:"code"`
	Name     string       `json:"name"`
	Icon     string       `json:"icon"`
	Total    int          `json:"total"`
	Counts   StatusCounts `json:"counts"`
	WeekNew  int          `json:"week_new"`
	Progress float64      `json:"progress"`
}

func (s *DashboardService) Subjects(childID int64) ([]SubjectSummary, error) {
	type row struct {
		Code   string
		Name   string
		Icon   string
		Status string
		N      int
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT s.code, s.name, s.icon, `+effectiveStatusSQL+` AS status, COUNT(1) AS n
		FROM subjects s
		JOIN modules m           ON m.subject_id = s.id
		JOIN knowledge_points kp ON kp.module_id = m.id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		GROUP BY s.code, s.name, s.icon, s.order_no, 4
		ORDER BY s.order_no`, childID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	order := []string{}
	acc := map[string]*SubjectSummary{}
	for _, r := range rows {
		cur, exists := acc[r.Code]
		if !exists {
			cur = &SubjectSummary{Code: r.Code, Name: r.Name, Icon: r.Icon}
			acc[r.Code] = cur
			order = append(order, r.Code)
		}
		cur.Total += r.N
		switch r.Status {
		case "not_started":
			cur.Counts.NotStarted += r.N
		case "learning":
			cur.Counts.Learning += r.N
		case "shaky":
			cur.Counts.Shaky += r.N
		case "mastered":
			cur.Counts.Mastered += r.N
		case "review_due":
			cur.Counts.ReviewDue += r.N
		}
	}

	weekAgo := time.Now().AddDate(0, 0, -7)
	type wrow struct {
		Code string
		N    int
	}
	var wrows []wrow
	_ = s.repo.DB().Raw(`
		SELECT s.code, COUNT(1) AS n
		FROM mastery_states ms
		JOIN knowledge_points kp ON kp.id = ms.kp_id
		JOIN modules m           ON m.id = kp.module_id
		JOIN subjects s          ON s.id = m.subject_id
		WHERE ms.child_id = ? AND ms.mastered_at IS NOT NULL AND ms.mastered_at >= ?
		GROUP BY s.code`, childID, weekAgo).Scan(&wrows).Error
	for _, w := range wrows {
		if cur, okk := acc[w.Code]; okk {
			cur.WeekNew = w.N
		}
	}

	out := make([]SubjectSummary, 0, len(order))
	for _, code := range order {
		cur := acc[code]
		if cur.Total > 0 {
			cur.Progress = float64(cur.Counts.Mastered+cur.Counts.ReviewDue) / float64(cur.Total)
		}
		out = append(out, *cur)
	}
	return out, nil
}

type MatrixSkill struct {
	Code     string  `json:"code"`
	Status   string  `json:"status"`
	Accuracy float64 `json:"accuracy"`
	Attempts int     `json:"attempts"`
}

type MatrixPoint struct {
	ID       int64         `json:"id"`
	Title    string        `json:"title"`
	Status   string        `json:"status"`
	Accuracy float64       `json:"accuracy"`
	Attempts int           `json:"attempts"`
	DueAt    *time.Time    `json:"due_at"`
	Skills   []MatrixSkill `json:"skills,omitempty"`
}

type MatrixModule struct {
	Code     string        `json:"code"`
	Name     string        `json:"name"`
	Total    int           `json:"total"`
	Mastered int           `json:"mastered"`
	Points   []MatrixPoint `json:"points"`
}

type Matrix struct {
	Subject SubjectSummary `json:"subject"`
	Modules []MatrixModule `json:"modules"`
}

func skillsFromRows(skillSet []string, byCode map[string]model.MasterySkill, now time.Time) (skills []MatrixSkill, rollup mastery.Status) {
	statuses := make([]mastery.Status, 0, len(skillSet))
	skills = make([]MatrixSkill, 0, len(skillSet))
	for _, code := range skillSet {
		row, ok := byCode[code]
		st := mastery.StatusNotStarted
		attempts, correct := 0, 0
		if ok {
			eng := skillToEngine(row)
			st = mastery.Display(eng, now)
			attempts, correct = eng.Attempts, eng.Correct
		}
		skillAcc := 0.0
		if attempts > 0 {
			skillAcc = float64(correct) / float64(attempts)
		}
		skills = append(skills, MatrixSkill{
			Code: code, Status: string(st),
			Accuracy: skillAcc, Attempts: attempts,
		})
		statuses = append(statuses, st)
	}
	return skills, mastery.RollupSkills(statuses)
}

func (s *DashboardService) Matrix(childID int64, subjectCode string) (Matrix, error) {
	var out Matrix

	subjects, err := s.Subjects(childID)
	if err != nil {
		return out, err
	}
	for _, sub := range subjects {
		if sub.Code == subjectCode {
			out.Subject = sub
		}
	}
	if out.Subject.Code == "" {
		return out, fmt.Errorf("未知学科 %q", subjectCode)
	}

	type row struct {
		ModuleCode string
		ModuleName string
		KpID       int64
		Title      string
		Status     string
		Attempts   int
		Correct    int
		DueAt      *time.Time
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT m.code AS module_code, m.name AS module_name,
		       kp.id AS kp_id, kp.title,
		       `+effectiveStatusSQL+` AS status,
		       COALESCE(ms.attempts,0) AS attempts,
		       COALESCE(ms.correct,0)  AS correct,
		       ms.due_at
		FROM subjects s
		JOIN modules m           ON m.subject_id = s.id
		JOIN knowledge_points kp ON kp.module_id = m.id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		WHERE s.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id`, childID, subjectCode).
		Scan(&rows).Error; err != nil {
		return out, err
	}

	idx := map[string]int{}
	kpIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		kpIDs = append(kpIDs, r.KpID)
	}

	skillByKp := map[int64]map[string]model.MasterySkill{}
	skillSet := mastery.SkillsForSubject(subjectCode)
	if len(skillSet) > 0 && len(kpIDs) > 0 {
		skillRows, err := s.repo.ListMasterySkills(s.repo.DB(), childID, kpIDs)
		if err != nil {
			return out, err
		}
		for _, sk := range skillRows {
			if skillByKp[sk.KpID] == nil {
				skillByKp[sk.KpID] = map[string]model.MasterySkill{}
			}
			skillByKp[sk.KpID][sk.SkillCode] = sk
		}
	}

	now := time.Now()
	for _, r := range rows {
		i, exists := idx[r.ModuleCode]
		if !exists {
			out.Modules = append(out.Modules, MatrixModule{Code: r.ModuleCode, Name: r.ModuleName})
			i = len(out.Modules) - 1
			idx[r.ModuleCode] = i
		}
		acc := 0.0
		if r.Attempts > 0 {
			acc = float64(r.Correct) / float64(r.Attempts)
		}

		status := r.Status
		var skills []MatrixSkill
		if len(skillSet) > 0 {
			var rollup mastery.Status
			skills, rollup = skillsFromRows(skillSet, skillByKp[r.KpID], now)
			status = string(rollup)
		}

		mod := &out.Modules[i]
		mod.Total++
		if status == "mastered" || status == "review_due" {
			mod.Mastered++
		}
		mod.Points = append(mod.Points, MatrixPoint{
			ID: r.KpID, Title: r.Title, Status: status,
			Accuracy: acc, Attempts: r.Attempts, DueAt: r.DueAt,
			Skills: skills,
		})
	}

	// 技能学科顶栏「完全掌握」与字卡状态一致：按技能 rollup 重算学科计数。
	if len(skillSet) > 0 {
		var counts StatusCounts
		for _, mod := range out.Modules {
			for _, p := range mod.Points {
				switch p.Status {
				case "not_started":
					counts.NotStarted++
				case "learning":
					counts.Learning++
				case "shaky":
					counts.Shaky++
				case "mastered":
					counts.Mastered++
				case "review_due":
					counts.ReviewDue++
				}
			}
		}
		out.Subject.Counts = counts
		if out.Subject.Total > 0 {
			out.Subject.Progress = float64(counts.Mastered+counts.ReviewDue) / float64(out.Subject.Total)
		}
	}
	return out, nil
}

type AttentionItem struct {
	KpID        int64      `json:"kp_id"`
	Title       string     `json:"title"`
	SubjectCode string     `json:"subject_code"`
	SubjectName string     `json:"subject_name"`
	ModuleName  string     `json:"module_name"`
	Status      string     `json:"status"`
	Accuracy    float64    `json:"accuracy"`
	WrongCount  int        `json:"wrong_count"`
	Attempts    int        `json:"attempts"`
	DueAt       *time.Time `json:"due_at"`
}

func (s *DashboardService) Attention(childID int64, limit int) ([]AttentionItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	var out []AttentionItem
	err := s.repo.DB().Raw(`
		SELECT ms.kp_id, kp.title,
		       s.code AS subject_code, s.name AS subject_name, m.name AS module_name,
		       `+effectiveStatusSQL+` AS status,
		       CASE WHEN ms.attempts > 0
		            THEN CAST(ms.correct AS REAL) / ms.attempts ELSE 0 END AS accuracy,
		       (ms.attempts - ms.correct) AS wrong_count,
		       ms.attempts, ms.due_at
		FROM mastery_states ms
		JOIN knowledge_points kp ON kp.id = ms.kp_id
		JOIN modules m           ON m.id = kp.module_id
		JOIN subjects s          ON s.id = m.subject_id
		WHERE ms.child_id = ?
		  AND (`+effectiveStatusSQL+`) IN ('shaky','review_due')
		ORDER BY CASE WHEN (`+effectiveStatusSQL+`) = 'shaky' THEN 0 ELSE 1 END,
		         accuracy ASC, wrong_count DESC, ms.due_at ASC
		LIMIT ?`, childID, limit).Scan(&out).Error
	return out, err
}

type HistoryItem struct {
	At        time.Time `json:"at"`
	IsCorrect bool      `json:"is_correct"`
	CostMs    int       `json:"cost_ms"`
	Source    string    `json:"source"`
	SkillCode string    `json:"skill_code,omitempty"`
}

type KpDetail struct {
	KpID        int64         `json:"kp_id"`
	Title       string        `json:"title"`
	Payload     string        `json:"payload"`
	Difficulty  int           `json:"difficulty"`
	SubjectCode string        `json:"subject_code"`
	SubjectName string        `json:"subject_name"`
	ModuleName  string        `json:"module_name"`
	Status      string        `json:"status"`
	Attempts    int           `json:"attempts"`
	Correct     int           `json:"correct"`
	Accuracy    float64       `json:"accuracy"`
	Streak      int           `json:"streak"`
	BestStreak  int           `json:"best_streak"`
	DueAt       *time.Time    `json:"due_at"`
	MasteredAt  *time.Time    `json:"mastered_at"`
	Skills      []MatrixSkill `json:"skills,omitempty" gorm:"-"`
	History     []HistoryItem `json:"history" gorm:"-"`
}

func (s *DashboardService) KpDetail(childID, kpID int64) (KpDetail, error) {
	var out KpDetail
	err := s.repo.DB().Raw(`
		SELECT kp.id AS kp_id, kp.title, kp.payload, kp.difficulty,
		       s.code AS subject_code, s.name AS subject_name, m.name AS module_name,
		       `+effectiveStatusSQL+` AS status,
		       COALESCE(ms.attempts,0) AS attempts, COALESCE(ms.correct,0) AS correct,
		       COALESCE(ms.streak,0) AS streak, COALESCE(ms.best_streak,0) AS best_streak,
		       ms.due_at, ms.mastered_at
		FROM knowledge_points kp
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		WHERE kp.id = ?`, childID, kpID).Scan(&out).Error
	if err != nil {
		return out, err
	}
	if out.KpID == 0 {
		return out, fmt.Errorf("知识点 %d 不存在", kpID)
	}
	if out.Attempts > 0 {
		out.Accuracy = float64(out.Correct) / float64(out.Attempts)
	}

	type historyRow struct {
		At           time.Time
		IsCorrect    bool
		CostMs       int
		Source       string
		QuestionCode string
	}
	var histRows []historyRow
	if err := s.repo.DB().Raw(`
		SELECT a.created_at AS at, a.is_correct, a.cost_ms, a.source, q.code AS question_code
		FROM attempts a
		LEFT JOIN questions q ON q.id = a.question_id
		WHERE a.child_id = ? AND a.kp_id = ?
		ORDER BY a.created_at ASC`, childID, kpID).Scan(&histRows).Error; err != nil {
		return out, err
	}
	out.History = make([]HistoryItem, 0, len(histRows))
	for _, r := range histRows {
		out.History = append(out.History, HistoryItem{
			At: r.At, IsCorrect: r.IsCorrect, CostMs: r.CostMs, Source: r.Source,
			SkillCode: mastery.SkillFromQuestionCode(r.QuestionCode),
		})
	}

	if skillSet := mastery.SkillsForSubject(out.SubjectCode); len(skillSet) > 0 {
		now := time.Now()
		skillByCode := map[string]model.MasterySkill{}
		skillRows, err := s.repo.ListMasterySkills(s.repo.DB(), childID, []int64{kpID})
		if err != nil {
			return out, err
		}
		for _, sk := range skillRows {
			skillByCode[sk.SkillCode] = sk
		}
		var rollup mastery.Status
		out.Skills, rollup = skillsFromRows(skillSet, skillByCode, now)
		out.Status = string(rollup)
	}

	return out, nil
}
