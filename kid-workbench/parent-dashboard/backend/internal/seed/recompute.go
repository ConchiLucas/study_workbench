package seed

import (
	"gorm.io/gorm"

	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
)

func Recompute(gdb *gorm.DB, cfg mastery.Config, childID int64) error {
	r := repo.New(gdb)
	return gdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("child_id = ?", childID).Delete(&model.MasteryState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("child_id = ?", childID).Delete(&model.DailyStat{}).Error; err != nil {
			return err
		}

		var rows []model.Attempt
		if err := tx.Where("child_id = ?", childID).
			Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
			return err
		}

		difficulty := map[int64]int{}
		states := map[int64]mastery.State{}

		for _, a := range rows {
			d, cached := difficulty[a.KpID]
			if !cached {
				kp, err := r.GetKnowledgePoint(tx, a.KpID)
				if err != nil {
					return err
				}
				d = kp.Difficulty
				difficulty[a.KpID] = d
			}

			cur, seen := states[a.KpID]
			if !seen {
				cur = mastery.NewState()
			}
			next := mastery.Apply(cur, mastery.Attempt{
				Correct: a.IsCorrect, At: a.CreatedAt, Source: a.Source,
			}, d, cfg)
			states[a.KpID] = next

			newly := 0
			if cur.MasteredAt == nil && next.MasteredAt != nil {
				newly = 1
			}
			reviewDone := 0
			if cur.Status == mastery.StatusMastered && a.IsCorrect {
				reviewDone = 1
			}
			if a.Source != mastery.SourceParentMark {
				if err := r.BumpDailyStat(tx, childID, a.CreatedAt, model.DailyStat{
					PracticeSec: maxInt(a.CostMs/1000, 1), Attempts: 1,
					Correct: boolToInt(a.IsCorrect), NewlyMastered: newly, ReviewDone: reviewDone,
				}); err != nil {
					return err
				}
			}
		}

		for kpID, st := range states {
			row := model.MasteryState{
				ChildID: childID, KpID: kpID, Status: string(st.Status),
				Attempts: st.Attempts, Correct: st.Correct, Streak: st.Streak,
				BestStreak: st.BestStreak, Ease: st.Ease, IntervalDays: st.IntervalDays,
				MasteredAt: st.MasteredAt,
			}
			if !st.DueAt.IsZero() {
				t := st.DueAt
				row.DueAt = &t
			}
			if !st.FirstSeenAt.IsZero() {
				t := st.FirstSeenAt
				row.FirstSeenAt = &t
			}
			if err := r.UpsertMastery(tx, &row); err != nil {
				return err
			}
		}
		return nil
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
