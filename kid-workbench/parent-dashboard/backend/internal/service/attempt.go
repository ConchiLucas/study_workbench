package service

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
)

type AttemptService struct {
	repo *repo.Repo
	cfg  mastery.Config
}

func NewAttemptService(r *repo.Repo, cfg mastery.Config) *AttemptService {
	return &AttemptService{repo: r, cfg: cfg}
}

type AttemptInput struct {
	ClientID   string    `json:"client_id" binding:"required"`
	KpID       int64     `json:"kp_id" binding:"required"`
	QuestionID *int64    `json:"question_id"`
	IsCorrect  bool      `json:"is_correct"`
	CostMs     int       `json:"cost_ms"`
	Source     string    `json:"source"`
	At         time.Time `json:"at"`
}

type StateDTO struct {
	KpID         int64      `json:"kp_id"`
	Status       string     `json:"status"`
	Streak       int        `json:"streak"`
	Attempts     int        `json:"attempts"`
	Accuracy     float64    `json:"accuracy"`
	IntervalDays int        `json:"interval_days"`
	DueAt        *time.Time `json:"due_at"`
}

func (s *AttemptService) Report(childID int64, in []AttemptInput) ([]StateDTO, error) {
	out := make([]StateDTO, 0, len(in))
	seen := map[int64]bool{}

	err := s.repo.Tx(func(tx *gorm.DB) error {
		for _, a := range in {
			dto, applied, err := s.ApplyOne(tx, childID, a)
			if err != nil {
				return err
			}
			if !applied {
				continue
			}

			if !seen[a.KpID] {
				seen[a.KpID] = true
				out = append(out, dto)
			} else {
				for i := range out {
					if out[i].KpID == a.KpID {
						out[i] = dto
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ApplyOne 在调用方的事务里落一次作答：写 attempts、重算掌握度、累加当日统计、
// 新掌握发小红花。答题计划走这个入口，好处是计划进度和掌握度在同一个事务里提交。
//
// applied 为 false 表示该 client_id 已经处理过（幂等重放），调用方应跳过。
func (s *AttemptService) ApplyOne(tx *gorm.DB, childID int64, a AttemptInput) (StateDTO, bool, error) {
	if a.At.IsZero() {
		a.At = time.Now()
	}
	if a.Source == "" {
		a.Source = mastery.SourceQuiz
	}

	kp, err := s.repo.GetKnowledgePoint(tx, a.KpID)
	if err != nil {
		return StateDTO{}, false, err
	}

	row := &model.Attempt{
		ChildID: childID, KpID: a.KpID, QuestionID: a.QuestionID,
		IsCorrect: a.IsCorrect, CostMs: a.CostMs,
		Source: a.Source, ClientID: a.ClientID, CreatedAt: a.At,
	}
	if err := s.repo.InsertAttempt(tx, row); err != nil {
		if isUniqueViolation(err) {
			return StateDTO{}, false, nil
		}
		return StateDTO{}, false, err
	}

	skillCode := ""
	if a.QuestionID != nil && *a.QuestionID > 0 {
		if q, qerr := s.repo.GetQuestion(tx, *a.QuestionID); qerr == nil {
			skillCode = mastery.SkillFromQuestionCode(q.Code)
		}
	}

	before, err := s.repo.GetMastery(tx, childID, a.KpID)
	if err != nil {
		return StateDTO{}, false, err
	}
	beforeEng := toEngine(before)

	subjectCode, err := s.subjectCodeForKp(tx, a.KpID)
	if err != nil {
		return StateDTO{}, false, err
	}
	skillSet := mastery.SkillsForSubject(subjectCode)

	var after mastery.State
	if len(skillSet) > 0 && (skillCode != "" || a.Source == mastery.SourceParentMark) {
		if err := s.applySubjectSkills(tx, childID, a, kp.Difficulty, skillCode, skillSet); err != nil {
			return StateDTO{}, false, err
		}
		after, err = s.rollupSubjectMastery(tx, childID, a.KpID, kp.Difficulty, skillSet)
		if err != nil {
			return StateDTO{}, false, err
		}
	} else {
		after = mastery.Apply(beforeEng, mastery.Attempt{
			Correct: a.IsCorrect, At: a.At, Source: a.Source,
		}, kp.Difficulty, s.cfg)
		next := fromEngine(childID, a.KpID, after)
		if err := s.repo.UpsertMastery(tx, &next); err != nil {
			return StateDTO{}, false, err
		}
	}

	newlyMastered := 0
	if beforeEng.MasteredAt == nil && after.MasteredAt != nil {
		newlyMastered = 1
	}
	// 识字：技能汇总刚变为「完全掌握」也算新掌握
	if newlyMastered == 0 &&
		mastery.Display(beforeEng, a.At) != mastery.StatusMastered &&
		mastery.Display(after, a.At) == mastery.StatusMastered {
		newlyMastered = 1
	}
	reviewDone := 0
	if beforeEng.Status == mastery.StatusMastered && a.IsCorrect {
		reviewDone = 1
	}

	if a.Source != mastery.SourceParentMark {
		if err := s.repo.BumpDailyStat(tx, childID, a.At, model.DailyStat{
			PracticeSec:   maxInt(a.CostMs/1000, 1),
			Attempts:      1,
			Correct:       boolToInt(a.IsCorrect),
			NewlyMastered: newlyMastered,
			ReviewDone:    reviewDone,
		}); err != nil {
			return StateDTO{}, false, err
		}
	}

	if newlyMastered == 1 {
		kpID := a.KpID
		if err := s.repo.AddFlowers(tx, childID, 1, "mastered", "knowledge_point", &kpID); err != nil {
			return StateDTO{}, false, err
		}
	}

	return toDTO(a.KpID, after), true, nil
}

// applySubjectSkills 更新某一技能（或家长一键该学科全部技能全过）。
func (s *AttemptService) applySubjectSkills(
	tx *gorm.DB, childID int64, a AttemptInput, difficulty int, skillCode string, skillSet []string,
) error {
	codes := []string{skillCode}
	if a.Source == mastery.SourceParentMark {
		codes = append([]string{}, skillSet...)
	}
	for _, code := range codes {
		if code == "" {
			continue
		}
		cur, err := s.repo.GetMasterySkill(tx, childID, a.KpID, code)
		if err != nil {
			return err
		}
		before := skillToEngine(cur)
		after := mastery.Apply(before, mastery.Attempt{
			Correct: a.IsCorrect, At: a.At, Source: a.Source,
		}, difficulty, s.cfg)
		next := skillFromEngine(childID, a.KpID, code, after)
		if err := s.repo.UpsertMasterySkill(tx, &next); err != nil {
			return err
		}
	}
	return nil
}

// rollupSubjectMastery 用技能展示态汇总字级 mastery_states。
func (s *AttemptService) rollupSubjectMastery(
	tx *gorm.DB, childID, kpID int64, difficulty int, skillSet []string,
) (mastery.State, error) {
	now := time.Now()
	statuses := make([]mastery.Status, 0, len(skillSet))
	var best mastery.State
	anyMasteredAt := false
	for _, code := range skillSet {
		row, err := s.repo.GetMasterySkill(tx, childID, kpID, code)
		if err != nil {
			return mastery.State{}, err
		}
		eng := skillToEngine(row)
		statuses = append(statuses, mastery.Display(eng, now))
		if eng.Attempts > best.Attempts {
			best = eng
		}
		if eng.MasteredAt != nil {
			anyMasteredAt = true
		}
	}
	rolled := mastery.RollupSkills(statuses)
	out := best
	if out.Ease == 0 {
		out = mastery.NewState()
	}
	out.Status = rolled
	if rolled == mastery.StatusMastered || rolled == mastery.StatusReviewDue {
		if !anyMasteredAt && out.MasteredAt == nil {
			t := now
			out.MasteredAt = &t
		}
	} else {
		// 未完全掌握时不保留「已掌握」时间戳，避免花数误判
		if rolled != mastery.StatusMastered && rolled != mastery.StatusReviewDue {
			out.MasteredAt = nil
		}
	}
	_ = difficulty
	next := fromEngine(childID, kpID, out)
	next.Status = string(rolled)
	if err := s.repo.UpsertMastery(tx, &next); err != nil {
		return mastery.State{}, err
	}
	return out, nil
}

func skillToEngine(m model.MasterySkill) mastery.State {
	st := mastery.State{
		Status: mastery.Status(m.Status), Attempts: m.Attempts, Correct: m.Correct,
		Streak: m.Streak, BestStreak: m.BestStreak, Ease: m.Ease,
		IntervalDays: m.IntervalDays, MasteredAt: m.MasteredAt,
	}
	if m.Ease == 0 {
		st.Ease = 2.5
	}
	if m.Status == "" || m.Status == "not_started" {
		st.Status = mastery.StatusNotStarted
	}
	if m.DueAt != nil {
		st.DueAt = *m.DueAt
	}
	if m.FirstSeenAt != nil {
		st.FirstSeenAt = *m.FirstSeenAt
	}
	return st
}

func skillFromEngine(childID, kpID int64, skill string, s mastery.State) model.MasterySkill {
	out := model.MasterySkill{
		ChildID: childID, KpID: kpID, SkillCode: skill, Status: string(s.Status),
		Attempts: s.Attempts, Correct: s.Correct, Streak: s.Streak,
		BestStreak: s.BestStreak, Ease: s.Ease, IntervalDays: s.IntervalDays,
		MasteredAt: s.MasteredAt,
	}
	if !s.DueAt.IsZero() {
		t := s.DueAt
		out.DueAt = &t
	}
	if !s.FirstSeenAt.IsZero() {
		t := s.FirstSeenAt
		out.FirstSeenAt = &t
	}
	return out
}

func (s *AttemptService) MarkMastered(childID, kpID int64) (StateDTO, error) {
	states, err := s.Report(childID, []AttemptInput{{
		ClientID: "mark-" + time.Now().Format("20060102150405.000000000"),
		KpID:     kpID, IsCorrect: true,
		Source: mastery.SourceParentMark, At: time.Now(),
	}})
	if err != nil {
		return StateDTO{}, err
	}
	if len(states) == 0 {
		return StateDTO{}, errors.New("标记未生效")
	}
	return states[0], nil
}

func (s *AttemptService) UndoMark(childID, kpID int64) (StateDTO, error) {
	var dto StateDTO
	err := s.repo.Tx(func(tx *gorm.DB) error {
		if err := tx.Where("child_id = ? AND kp_id = ? AND source = ?",
			childID, kpID, mastery.SourceParentMark).Delete(&model.Attempt{}).Error; err != nil {
			return err
		}

		kp, err := s.repo.GetKnowledgePoint(tx, kpID)
		if err != nil {
			return err
		}
		rows, err := s.repo.ListAttemptsByKp(tx, childID, kpID)
		if err != nil {
			return err
		}

		history := make([]mastery.Attempt, 0, len(rows))
		for _, r := range rows {
			history = append(history, mastery.Attempt{Correct: r.IsCorrect, At: r.CreatedAt, Source: r.Source})
		}

		st := mastery.Replay(history, kp.Difficulty, s.cfg)
		next := fromEngine(childID, kpID, st)
		if err := s.repo.UpsertMastery(tx, &next); err != nil {
			return err
		}
		dto = toDTO(kpID, st)
		return nil
	})
	return dto, err
}

func toEngine(m model.MasteryState) mastery.State {
	st := mastery.State{
		Status: mastery.Status(m.Status), Attempts: m.Attempts, Correct: m.Correct,
		Streak: m.Streak, BestStreak: m.BestStreak, Ease: m.Ease,
		IntervalDays: m.IntervalDays, MasteredAt: m.MasteredAt,
	}
	if m.Ease == 0 {
		st.Ease = 2.5
	}
	if m.DueAt != nil {
		st.DueAt = *m.DueAt
	}
	if m.FirstSeenAt != nil {
		st.FirstSeenAt = *m.FirstSeenAt
	}
	return st
}

func fromEngine(childID, kpID int64, s mastery.State) model.MasteryState {
	out := model.MasteryState{
		ChildID: childID, KpID: kpID, Status: string(s.Status),
		Attempts: s.Attempts, Correct: s.Correct, Streak: s.Streak,
		BestStreak: s.BestStreak, Ease: s.Ease, IntervalDays: s.IntervalDays,
		MasteredAt: s.MasteredAt,
	}
	if !s.DueAt.IsZero() {
		t := s.DueAt
		out.DueAt = &t
	}
	if !s.FirstSeenAt.IsZero() {
		t := s.FirstSeenAt
		out.FirstSeenAt = &t
	}
	return out
}

func toDTO(kpID int64, s mastery.State) StateDTO {
	dto := StateDTO{
		KpID: kpID, Status: string(mastery.Display(s, time.Now())),
		Streak: s.Streak, Attempts: s.Attempts,
		Accuracy: s.Accuracy(), IntervalDays: s.IntervalDays,
	}
	if !s.DueAt.IsZero() {
		t := s.DueAt
		dto.DueAt = &t
	}
	return dto
}

func (s *AttemptService) subjectCodeForKp(tx *gorm.DB, kpID int64) (string, error) {
	var code string
	err := tx.Raw(`
		SELECT s.code FROM subjects s
		JOIN modules m ON m.subject_id = s.id
		JOIN knowledge_points kp ON kp.module_id = m.id
		WHERE kp.id = ?
	`, kpID).Scan(&code).Error
	return code, err
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
