package seed

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
)

type kpRow struct {
	ID          int64
	Difficulty  int
	SubjectCode string
}

// Demo 为每个学科都造学习记录：覆盖 mastered / learning / shaky / review_due / not_started。
func Demo(gdb *gorm.DB, cfg mastery.Config, childID int64, days int) error {
	if err := ResetLearningData(gdb, childID); err != nil {
		return err
	}

	svc := service.NewAttemptService(repo.New(gdb), cfg)
	rng := rand.New(rand.NewSource(20260822))

	var kps []kpRow
	if err := gdb.Raw(`
		SELECT kp.id, kp.difficulty, s.code AS subject_code
		FROM knowledge_points kp
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		ORDER BY s.order_no, m.order_no, kp.order_no, kp.id`).Scan(&kps).Error; err != nil {
		return err
	}
	if len(kps) == 0 {
		return fmt.Errorf("请先执行 catalog 种子")
	}

	bySubject := map[string][]kpRow{}
	order := []string{}
	for _, k := range kps {
		if _, ok := bySubject[k.SubjectCode]; !ok {
			order = append(order, k.SubjectCode)
		}
		bySubject[k.SubjectCode] = append(bySubject[k.SubjectCode], k)
	}

	// 每个学科选一组「会练到」的知识点（约 40%），其余保持未开始
	pool := make([]kpRow, 0, 280)
	for _, code := range order {
		list := bySubject[code]
		n := len(list)*2/5 + 4
		if n > len(list) {
			n = len(list)
		}
		if n < 3 && len(list) >= 3 {
			n = 3
		}
		shuffled := append([]kpRow(nil), list...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		pool = append(pool, shuffled[:n]...)
	}

	start := time.Now().AddDate(0, 0, -days)
	for d := 0; d < days; d++ {
		day := start.AddDate(0, 0, d)
		if rng.Float64() < 0.15 {
			continue
		}

		batch := make([]service.AttemptInput, 0, 40)
		subjectsToday := 3 + rng.Intn(3)
		for si := 0; si < subjectsToday; si++ {
			code := order[(d+si)%len(order)]
			list := bySubject[code]
			candidates := filterSubject(pool, code)
			if len(candidates) == 0 {
				candidates = list
			}
			pickN := 1 + rng.Intn(3)
			for i := 0; i < pickN; i++ {
				kp := candidates[rng.Intn(len(candidates))]
				pCorrect := 0.88 - 0.12*float64(kp.Difficulty-1)
				if rng.Float64() < 0.12 {
					pCorrect = 0.35
				}
				rounds := 1 + rng.Intn(3)
				for k := 0; k < rounds; k++ {
					at := day.Add(time.Duration(8+si*2+i) * time.Hour).
						Add(time.Duration(k*4) * time.Minute)
					batch = append(batch, service.AttemptInput{
						ClientID:  fmt.Sprintf("demo-%d-%d-%s-%d-%d", childID, d, code, i, k),
						KpID:      kp.ID,
						IsCorrect: rng.Float64() < pCorrect,
						CostMs:    1800 + rng.Intn(7000),
						Source:    mastery.SourceQuiz,
						At:        at,
					})
				}
			}
		}
		if _, err := svc.Report(childID, batch); err != nil {
			return err
		}
	}

	return forceSomeReviewDue(gdb, childID)
}

// ResetLearningData 清空该孩子的作答/掌握/日统计/小红花流水，便于重灌 demo
func ResetLearningData(gdb *gorm.DB, childID int64) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, sql := range []string{
			`DELETE FROM attempts WHERE child_id = ?`,
			`DELETE FROM mastery_states WHERE child_id = ?`,
			`DELETE FROM daily_stats WHERE child_id = ?`,
			`DELETE FROM flower_ledger WHERE child_id = ?`,
			`DELETE FROM study_sessions WHERE child_id = ?`,
		} {
			if err := tx.Exec(sql, childID).Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.Child{}).Where("id = ?", childID).Update("flowers", 0).Error
	})
}

func filterSubject(pool []kpRow, code string) []kpRow {
	out := make([]kpRow, 0, 16)
	for _, k := range pool {
		if k.SubjectCode == code {
			out = append(out, k)
		}
	}
	return out
}

func forceSomeReviewDue(gdb *gorm.DB, childID int64) error {
	type row struct {
		KpID      int64
		SubjectID int64
	}
	var rows []row
	if err := gdb.Raw(`
		SELECT ms.kp_id, s.id AS subject_id
		FROM mastery_states ms
		JOIN knowledge_points kp ON kp.id = ms.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE ms.child_id = ? AND ms.status = 'mastered'
		ORDER BY s.id, ms.kp_id`, childID).Scan(&rows).Error; err != nil {
		return err
	}

	perSubject := map[int64]int{}
	ids := make([]int64, 0, 24)
	for _, r := range rows {
		if perSubject[r.SubjectID] >= 3 {
			continue
		}
		perSubject[r.SubjectID]++
		ids = append(ids, r.KpID)
	}
	if len(ids) == 0 {
		return nil
	}

	due := time.Now().Add(-48 * time.Hour)
	return gdb.Model(&model.MasteryState{}).
		Where("child_id = ? AND kp_id IN ?", childID, ids).
		Update("due_at", due).Error
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
