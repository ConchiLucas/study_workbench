package service_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/seed"
	"github.com/conchi/study-workbench/internal/service"
)

func newAttemptSvc(t *testing.T) (*service.AttemptService, *gorm.DB) {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	return service.NewAttemptService(repo.New(gdb), mastery.DefaultConfig()), gdb
}

func mathKpID(t *testing.T, gdb *gorm.DB) int64 {
	t.Helper()
	var id int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'math' AND m.code = 'add10'
		ORDER BY kp.order_no LIMIT 1`).Scan(&id).Error)
	require.NotZero(t, id)
	return id
}

func TestReportAttemptsUpdatesMasteryAndStats(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	kpID := mathKpID(t, gdb)
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

	in := []service.AttemptInput{
		{ClientID: "c1", KpID: kpID, IsCorrect: true, CostMs: 3000, Source: "quiz", At: now},
		{ClientID: "c2", KpID: kpID, IsCorrect: true, CostMs: 2500, Source: "quiz", At: now.Add(time.Minute)},
		{ClientID: "c3", KpID: kpID, IsCorrect: true, CostMs: 2000, Source: "quiz", At: now.Add(2 * time.Minute)},
	}
	states, err := svc.Report(1, in)
	require.NoError(t, err)
	require.Len(t, states, 1)
	require.Contains(t, []string{string(mastery.StatusMastered), string(mastery.StatusReviewDue)}, states[0].Status)

	var ms model.MasteryState
	require.NoError(t, gdb.Where("child_id = 1 AND kp_id = ?", kpID).First(&ms).Error)
	require.Equal(t, string(mastery.StatusMastered), ms.Status)

	var ds model.DailyStat
	require.NoError(t, gdb.Where("child_id = 1 AND stat_date = ?", "2026-08-20").First(&ds).Error)
	require.Equal(t, 3, ds.Attempts)
	require.Equal(t, 3, ds.Correct)
	require.Equal(t, 1, ds.NewlyMastered)
	require.Greater(t, ds.PracticeSec, 0)

	var child model.Child
	require.NoError(t, gdb.First(&child, 1).Error)
	require.Equal(t, 1, child.Flowers)
}

func TestReportIsIdempotent(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	kpID := mathKpID(t, gdb)
	now := time.Now()

	in := []service.AttemptInput{
		{ClientID: "dup", KpID: kpID, IsCorrect: true, Source: "quiz", At: now},
	}
	_, err := svc.Report(1, in)
	require.NoError(t, err)
	_, err = svc.Report(1, in)
	require.NoError(t, err)

	var attempts int64
	require.NoError(t, gdb.Model(&model.Attempt{}).Count(&attempts).Error)
	require.Equal(t, int64(1), attempts)

	var st model.MasteryState
	require.NoError(t, gdb.Where("child_id = 1 AND kp_id = ?", kpID).First(&st).Error)
	require.Equal(t, 1, st.Attempts)
}

func TestMarkMasteredAndUndo(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	kpID := mathKpID(t, gdb)
	now := time.Now()

	_, err := svc.Report(1, []service.AttemptInput{
		{ClientID: "a1", KpID: kpID, IsCorrect: true, Source: "quiz", At: now},
		{ClientID: "a2", KpID: kpID, IsCorrect: false, Source: "quiz", At: now.Add(time.Minute)},
	})
	require.NoError(t, err)

	st, err := svc.MarkMastered(1, kpID)
	require.NoError(t, err)
	require.Equal(t, string(mastery.StatusMastered), st.Status)

	st, err = svc.UndoMark(1, kpID)
	require.NoError(t, err)
	require.Equal(t, string(mastery.StatusLearning), st.Status)
	require.Equal(t, 2, st.Attempts)
}

func newDashboard(t *testing.T) (*service.DashboardService, *gorm.DB) {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, seed.Demo(gdb, mastery.DefaultConfig(), 1, 60))
	return service.NewDashboardService(repo.New(gdb)), gdb
}

func TestOverviewCountsAddUpToTotal(t *testing.T) {
	svc, _ := newDashboard(t)
	ov, err := svc.Overview(1)
	require.NoError(t, err)

	require.Equal(t, 668, ov.TotalKp)
	sum := ov.Counts.Mastered + ov.Counts.Learning + ov.Counts.Shaky +
		ov.Counts.ReviewDue + ov.Counts.NotStarted
	require.Equal(t, ov.TotalKp, sum)
	require.Greater(t, ov.Counts.Mastered, 0)
	require.Equal(t, "卢沁一", ov.Child.Name)
}

func TestSubjectsSummary(t *testing.T) {
	svc, _ := newDashboard(t)
	list, err := svc.Subjects(1)
	require.NoError(t, err)
	require.Len(t, list, 8)

	byCode := map[string]service.SubjectSummary{}
	for _, s := range list {
		byCode[s.Code] = s
		require.Equal(t, s.Total, s.Counts.Mastered+s.Counts.Learning+
			s.Counts.Shaky+s.Counts.ReviewDue+s.Counts.NotStarted)
	}
	require.Equal(t, 74, byCode["math"].Total)
	require.Equal(t, 200, byCode["english"].Total)
}

func TestMatrixGroupsByModule(t *testing.T) {
	svc, _ := newDashboard(t)
	m, err := svc.Matrix(1, "math")
	require.NoError(t, err)

	require.Equal(t, "算术", m.Subject.Name)
	require.Equal(t, 74, m.Subject.Total)
	require.Len(t, m.Modules, 5)

	total := 0
	for _, mod := range m.Modules {
		require.NotEmpty(t, mod.Name)
		require.Len(t, mod.Points, mod.Total)
		total += len(mod.Points)
		for _, p := range mod.Points {
			require.NotEmpty(t, p.Title)
			require.Contains(t,
				[]string{"not_started", "learning", "shaky", "mastered", "review_due"}, p.Status)
		}
	}
	require.Equal(t, 74, total)
}

func TestAttentionRanksWorstFirst(t *testing.T) {
	svc, _ := newDashboard(t)
	list, err := svc.Attention(1, 10)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	require.LessOrEqual(t, len(list), 10)

	for i := range list {
		require.Contains(t, []string{"shaky", "review_due"}, list[i].Status)
		require.NotEmpty(t, list[i].SubjectName)
	}
}

func TestKpDetailIncludesHistory(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp_id FROM attempts WHERE child_id = 1
		GROUP BY kp_id ORDER BY COUNT(1) DESC LIMIT 1`).Scan(&kpID).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	require.Equal(t, kpID, d.KpID)
	require.NotEmpty(t, d.Title)
	require.NotEmpty(t, d.History)
	require.Equal(t, len(d.History), d.Attempts)
}

func TestKpDetailLiteracyIncludesSkills(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy'
		ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)
	require.NotZero(t, kpID)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	require.Equal(t, "literacy", d.SubjectCode)
	require.Len(t, d.Skills, 2)
	require.Equal(t, mastery.LiteracySkills, []string{d.Skills[0].Code, d.Skills[1].Code})
	for _, sk := range d.Skills {
		require.Contains(t, []string{"not_started", "learning", "shaky", "mastered", "review_due"}, sk.Status)
	}
}

func TestKpDetailLiteracyNoSkillRowsNotStarted(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy'
		ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)
	require.NotZero(t, kpID)
	require.NoError(t, gdb.Where("child_id = ? AND kp_id = ?", 1, kpID).Delete(&model.MasterySkill{}).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	require.Equal(t, "not_started", d.Status)
	require.Len(t, d.Skills, 2)
	for _, sk := range d.Skills {
		require.Equal(t, "not_started", sk.Status)
	}
}

func TestKpDetailHistorySkillCodeFromQuestion(t *testing.T) {
	svc, gdb := newDashboard(t)

	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy'
		ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)
	require.NotZero(t, kpID)

	q := model.Question{
		KpID: kpID, Code: mastery.SkillGlyphSense, Type: "choice",
		Stem: "test", Options: "[]", Answer: "0",
	}
	require.NoError(t, gdb.Create(&q).Error)

	qID := q.ID
	require.NoError(t, gdb.Create(&model.Attempt{
		ChildID: 1, KpID: kpID, QuestionID: &qID,
		IsCorrect: true, CostMs: 1200, Source: "quiz",
		ClientID: fmt.Sprintf("test-skill-%d", qID),
	}).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)

	var found bool
	for _, h := range d.History {
		if h.SkillCode == mastery.SkillGlyphSense {
			found = true
			require.True(t, h.IsCorrect)
		}
	}
	require.True(t, found, "expected history row with skill_code=glyph_sense")
}

func TestKpDetailParentMarkHasEmptySkillCode(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy' ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)

	require.NoError(t, gdb.Create(&model.Attempt{
		ChildID: 1, KpID: kpID, QuestionID: nil,
		IsCorrect: true, CostMs: 0, Source: "parent_mark",
		ClientID: fmt.Sprintf("parent-mark-%d", kpID),
	}).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	var sawMark bool
	for _, h := range d.History {
		if h.Source == "parent_mark" {
			sawMark = true
			require.Empty(t, h.SkillCode)
		}
	}
	require.True(t, sawMark)
}

func newStats(t *testing.T) *service.StatsService {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, seed.Demo(gdb, mastery.DefaultConfig(), 1, 60))
	return service.NewStatsService(repo.New(gdb))
}

func TestTrendFillsEmptyDays(t *testing.T) {
	svc := newStats(t)
	pts, err := svc.Trend(1, 30)
	require.NoError(t, err)
	require.Len(t, pts, 30)

	cumulative := 0
	for _, p := range pts {
		require.NotEmpty(t, p.Date)
		require.GreaterOrEqual(t, p.CumulativeMastered, cumulative)
		cumulative = p.CumulativeMastered
	}
	require.Greater(t, cumulative, 0)
}

func TestCalendarReturnsActiveDays(t *testing.T) {
	svc := newStats(t)
	days, err := svc.Calendar(1, 3)
	require.NoError(t, err)
	require.NotEmpty(t, days)
}

func TestReviewQueuePrioritisesShaky(t *testing.T) {
	svc := newStats(t)
	q, err := svc.ReviewQueue(1, 20)
	require.NoError(t, err)
	require.NotEmpty(t, q)
	require.Equal(t, "shaky", q[0].Status)
}

func TestRedeemRewardDeductsFlowers(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, gdb.Create(&model.Reward{ChildID: 1, Name: "看动画片 20 分钟", Cost: 5, Stock: 3}).Error)
	require.NoError(t, gdb.Model(&model.Child{}).Where("id = 1").Update("flowers", 6).Error)

	svc := service.NewRewardService(repo.New(gdb))
	require.NoError(t, svc.Redeem(1, 1))

	var child model.Child
	require.NoError(t, gdb.First(&child, 1).Error)
	require.Equal(t, 1, child.Flowers)

	require.Error(t, svc.Redeem(1, 1))
}
