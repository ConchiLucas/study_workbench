package repo

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/conchi/study-workbench/internal/model"
)

type Repo struct{ db *gorm.DB }

func New(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) DB() *gorm.DB { return r.db }

func (r *Repo) Tx(fn func(tx *gorm.DB) error) error { return r.db.Transaction(fn) }

func (r *Repo) GetMastery(tx *gorm.DB, childID, kpID int64) (model.MasteryState, error) {
	var st model.MasteryState
	err := tx.Where("child_id = ? AND kp_id = ?", childID, kpID).First(&st).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.MasteryState{ChildID: childID, KpID: kpID, Status: "not_started", Ease: 2.5}, nil
	}
	return st, err
}

func (r *Repo) UpsertMastery(tx *gorm.DB, st *model.MasteryState) error {
	st.UpdatedAt = time.Now()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "child_id"}, {Name: "kp_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "attempts", "correct", "streak", "best_streak", "ease",
			"interval_days", "due_at", "first_seen_at", "mastered_at", "updated_at",
		}),
	}).Create(st).Error
}

func (r *Repo) GetMasterySkill(tx *gorm.DB, childID, kpID int64, skill string) (model.MasterySkill, error) {
	var st model.MasterySkill
	err := tx.Where("child_id = ? AND kp_id = ? AND skill_code = ?", childID, kpID, skill).First(&st).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.MasterySkill{
			ChildID: childID, KpID: kpID, SkillCode: skill,
			Status: "not_started", Ease: 2.5,
		}, nil
	}
	return st, err
}

func (r *Repo) UpsertMasterySkill(tx *gorm.DB, st *model.MasterySkill) error {
	st.UpdatedAt = time.Now()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "child_id"}, {Name: "kp_id"}, {Name: "skill_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "attempts", "correct", "streak", "best_streak", "ease",
			"interval_days", "due_at", "first_seen_at", "mastered_at", "updated_at",
		}),
	}).Create(st).Error
}

func (r *Repo) ListMasterySkills(tx *gorm.DB, childID int64, kpIDs []int64) ([]model.MasterySkill, error) {
	var out []model.MasterySkill
	if len(kpIDs) == 0 {
		return out, nil
	}
	err := tx.Where("child_id = ? AND kp_id IN ?", childID, kpIDs).Find(&out).Error
	return out, err
}

func (r *Repo) GetQuestion(tx *gorm.DB, id int64) (model.Question, error) {
	var q model.Question
	err := tx.First(&q, id).Error
	return q, err
}

func (r *Repo) GetKnowledgePoint(tx *gorm.DB, kpID int64) (model.KnowledgePoint, error) {
	var kp model.KnowledgePoint
	err := tx.First(&kp, kpID).Error
	return kp, err
}

func (r *Repo) InsertAttempt(tx *gorm.DB, a *model.Attempt) error {
	return tx.Create(a).Error
}

func (r *Repo) ListAttemptsByKp(tx *gorm.DB, childID, kpID int64) ([]model.Attempt, error) {
	var out []model.Attempt
	err := tx.Where("child_id = ? AND kp_id = ?", childID, kpID).
		Order("created_at ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *Repo) BumpDailyStat(tx *gorm.DB, childID int64, day time.Time, delta model.DailyStat) error {
	d := day.Format("2006-01-02")
	return tx.Exec(`
		INSERT INTO daily_stats (child_id, stat_date, practice_sec, attempts, correct, newly_mastered, review_done, checked_in)
		VALUES (?, ?, ?, ?, ?, ?, ?, TRUE)
		ON CONFLICT(child_id, stat_date) DO UPDATE SET
			practice_sec   = daily_stats.practice_sec   + EXCLUDED.practice_sec,
			attempts       = daily_stats.attempts       + EXCLUDED.attempts,
			correct        = daily_stats.correct        + EXCLUDED.correct,
			newly_mastered = daily_stats.newly_mastered + EXCLUDED.newly_mastered,
			review_done    = daily_stats.review_done    + EXCLUDED.review_done,
			checked_in     = TRUE`,
		childID, d, delta.PracticeSec, delta.Attempts, delta.Correct,
		delta.NewlyMastered, delta.ReviewDone).Error
}

func (r *Repo) AddFlowers(tx *gorm.DB, childID int64, delta int, reason string, refType string, refID *int64) error {
	if err := tx.Create(&model.FlowerLedger{
		ChildID: childID, Delta: delta, Reason: reason,
		RefType: refType, RefID: refID, CreatedAt: time.Now(),
	}).Error; err != nil {
		return err
	}
	return tx.Model(&model.Child{}).Where("id = ?", childID).
		UpdateColumn("flowers", gorm.Expr("flowers + ?", delta)).Error
}
