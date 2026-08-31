package service_test

import (
	"encoding/json"
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

func newPlanSvc(t *testing.T) (*service.PlanService, *gorm.DB) {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	_, err = seed.Questions(gdb)
	require.NoError(t, err)

	r := repo.New(gdb)
	attempts := service.NewAttemptService(r, mastery.DefaultConfig())
	return service.NewPlanService(r, attempts), gdb
}

// correctIndexFor 从库里读出正确答案下标，模拟"孩子答对了"。
func correctIndexFor(t *testing.T, gdb *gorm.DB, questionID int64) int {
	t.Helper()
	var raw string
	require.NoError(t, gdb.Raw(`SELECT answer FROM questions WHERE id = ?`, questionID).Scan(&raw).Error)
	var a struct{ Index int }
	require.NoError(t, json.Unmarshal([]byte(raw), &a))
	return a.Index
}

func TestTodayPlanHasTenQuestionsAcrossAtMostThreeSubjects(t *testing.T) {
	svc, _ := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	require.Equal(t, 10, detail.Plan.TargetCount)
	require.Len(t, detail.Items, 10)
	require.Equal(t, "pending", detail.Plan.Status)

	subjects := map[string]int{}
	for i, item := range detail.Items {
		require.Equal(t, i+1, item.Seq, "题目序号应该连续")
		require.Equal(t, "pending", item.Status)
		require.NotZero(t, item.Question.ID)
		require.NotEmpty(t, item.Question.Stem)
		subjects[item.SubjectCode]++
	}
	require.LessOrEqual(t, len(subjects), 3, "一份计划最多 3 个学科")
	for code, n := range subjects {
		require.LessOrEqual(t, n, 5, "%s 超过单科上限", code)
	}
}

func TestPlanNeverIncludesSubjectsWithoutQuestions(t *testing.T) {
	svc, _ := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)

	disabled := map[string]bool{"game": true}
	for _, item := range detail.Items {
		require.False(t, disabled[item.SubjectCode],
			"%s 没有题库，不该进入计划", item.SubjectCode)
	}
}

func TestPlanDoesNotLeakAnswers(t *testing.T) {
	svc, _ := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)

	// 下发的题目 JSON 里不能出现答案字段，否则孩子能在开发者工具里翻到答案。
	blob, err := json.Marshal(detail)
	require.NoError(t, err)
	require.NotContains(t, string(blob), `"answer"`)
	require.NotContains(t, string(blob), `"index"`)
}

func TestTodayIsIdempotentAndResumable(t *testing.T) {
	svc, _ := newPlanSvc(t)

	first, err := svc.Today(1)
	require.NoError(t, err)

	second, err := svc.Today(1)
	require.NoError(t, err)

	require.Equal(t, first.Plan.ID, second.Plan.ID, "同一天不该生成第二份主计划")
	require.Len(t, second.Items, len(first.Items))
	for i := range first.Items {
		require.Equal(t, first.Items[i].ID, second.Items[i].ID, "刷新页面后题目必须还是同一批")
		require.Equal(t, first.Items[i].Question.ID, second.Items[i].Question.ID)
	}
}

func TestCorrectAnswerAdvancesPlanAndMastery(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	item := detail.Items[0]

	res, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{
		OptionIndex: correctIndexFor(t, gdb, item.Question.ID),
		CostMs:      4200,
	})
	require.NoError(t, err)
	require.True(t, res.Correct)
	require.False(t, res.CanRetry)
	require.Equal(t, "correct", res.Status)
	require.Equal(t, 1, res.Plan.DoneCount)
	require.Equal(t, 1, res.Plan.CorrectCount)
	require.Equal(t, "doing", res.Plan.Status)

	// 作答必须落进 attempts，家长看板才看得到。
	var n int64
	require.NoError(t, gdb.Raw(
		`SELECT COUNT(1) FROM attempts WHERE child_id = 1 AND kp_id = ?`, item.KpID).Scan(&n).Error)
	require.EqualValues(t, 1, n)

	var ms model.MasteryState
	require.NoError(t, gdb.Where("child_id = 1 AND kp_id = ?", item.KpID).First(&ms).Error)
	require.Equal(t, 1, ms.Attempts)
	require.Equal(t, 1, ms.Correct)

	// 当日统计也要跟着涨。
	var stat model.DailyStat
	require.NoError(t, gdb.Where("child_id = 1").First(&stat).Error)
	require.Equal(t, 1, stat.Attempts)
	require.Equal(t, 4, stat.PracticeSec)
}

func TestWrongAnswerAllowsExactlyOneRetry(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	item := detail.Items[0]
	correct := correctIndexFor(t, gdb, item.Question.ID)
	wrong := (correct + 1) % 4

	first, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: wrong, CostMs: 5000})
	require.NoError(t, err)
	require.False(t, first.Correct)
	require.True(t, first.CanRetry, "第一次答错应该还能再试一次")
	require.Equal(t, "pending", first.Status)
	require.Equal(t, 1, first.Tries)
	require.Equal(t, 0, first.Plan.DoneCount, "题目还没结束，不该计入已完成")
	require.Equal(t, correct, first.AnswerIndex)

	second, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: wrong, CostMs: 5000})
	require.NoError(t, err)
	require.False(t, second.Correct)
	require.False(t, second.CanRetry, "两次机会用完就该给答案跳过")
	require.Equal(t, "wrong", second.Status)
	require.Equal(t, 2, second.Tries)
	require.Equal(t, 1, second.Plan.DoneCount)
	require.Equal(t, 0, second.Plan.CorrectCount)

	// 两次错误都要如实记进 attempts，掌握度才反映真实情况。
	var n int64
	require.NoError(t, gdb.Raw(
		`SELECT COUNT(1) FROM attempts WHERE child_id = 1 AND kp_id = ? AND is_correct = ?`,
		item.KpID, false).Scan(&n).Error)
	require.EqualValues(t, 2, n)
}

func TestRetryAfterWrongCanStillBeCorrect(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	item := detail.Items[0]
	correct := correctIndexFor(t, gdb, item.Question.ID)

	_, err = svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: (correct + 1) % 4})
	require.NoError(t, err)

	res, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: correct})
	require.NoError(t, err)
	require.True(t, res.Correct)
	require.Equal(t, "correct", res.Status)
	require.Equal(t, 1, res.Plan.CorrectCount)
}

func TestRetryRecordsPicks(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	item := detail.Items[0]
	correct := correctIndexFor(t, gdb, item.Question.ID)
	wrong := (correct + 1) % 4

	_, err = svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: wrong})
	require.NoError(t, err)
	_, err = svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: correct})
	require.NoError(t, err)

	var picks string
	require.NoError(t, gdb.Raw(`SELECT picks FROM plan_items WHERE id = ?`, item.ID).Scan(&picks).Error)
	require.Equal(t, fmt.Sprintf("%d,%d", wrong, correct), picks)
}

func TestAnsweringFinishedItemIsIdempotent(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	item := detail.Items[0]
	correct := correctIndexFor(t, gdb, item.Question.ID)

	_, err = svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: correct})
	require.NoError(t, err)

	// 孩子连点或网络重试：结果照样返回，但不能重复计分。
	again, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: correct})
	require.NoError(t, err)
	require.True(t, again.Correct)
	require.Equal(t, 1, again.Plan.DoneCount)
	require.Equal(t, 1, again.Plan.CorrectCount)

	var n int64
	require.NoError(t, gdb.Raw(
		`SELECT COUNT(1) FROM attempts WHERE child_id = 1 AND kp_id = ?`, item.KpID).Scan(&n).Error)
	require.EqualValues(t, 1, n, "重复提交不该再写一条作答")
}

func TestFinishAwardsStarsAndFlowersOnce(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	require.Zero(t, detail.Plan.Flowers, "没做完的计划不该显示已得的花")
	_, err = svc.Start(1, detail.Plan.ID)
	require.NoError(t, err)

	for _, item := range detail.Items {
		_, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{
			OptionIndex: correctIndexFor(t, gdb, item.Question.ID), CostMs: 3000,
		})
		require.NoError(t, err)
	}

	var before model.Child
	require.NoError(t, gdb.First(&before, 1).Error)

	res, err := svc.Finish(1, detail.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, 3, res.Stars, "全对应该是三星")
	require.Equal(t, 4, res.Flowers)
	require.Equal(t, "done", res.Plan.Status)
	// 用时取各题耗时之和（10 题 × 3 秒），而不是墙上时间。
	require.Equal(t, 30, res.Plan.DurationSec)

	// 重复结算不能再发一次花，但计划上的总数要照旧——
	// 结算页刷新后还得显示"得到 4 朵"。
	again, err := svc.Finish(1, detail.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, 3, again.Stars)
	require.Zero(t, again.Flowers, "Flowers 是这次新发的花")
	require.Equal(t, 4, again.Plan.Flowers, "Plan.Flowers 是这份计划拿到的总数")

	var afterLedger int64
	require.NoError(t, gdb.Raw(
		`SELECT COUNT(1) FROM flower_ledger WHERE child_id = 1 AND reason = 'plan_done'`).
		Scan(&afterLedger).Error)
	require.EqualValues(t, 1, afterLedger)
	_ = before
}

func TestStarsDropWithLowAccuracy(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)

	// 前 5 题答对，后 5 题两次都答错 → 正确率 50%，一星。
	for i, item := range detail.Items {
		correct := correctIndexFor(t, gdb, item.Question.ID)
		if i < 5 {
			_, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: correct})
			require.NoError(t, err)
			continue
		}
		for try := 0; try < 2; try++ {
			_, err := svc.Answer(1, detail.Plan.ID, item.ID,
				service.AnswerInput{OptionIndex: (correct + 1) % 4})
			require.NoError(t, err)
		}
	}

	res, err := svc.Finish(1, detail.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, 1, res.Stars)
	require.Equal(t, 2, res.Flowers, "一星也该有奖励")
	require.Equal(t, 10, res.Plan.DoneCount)
	require.Equal(t, 5, res.Plan.CorrectCount)
}

func TestPlanPrioritisesShakyAndReviewOverNew(t *testing.T) {
	svc, gdb := newPlanSvc(t)
	r := repo.New(gdb)
	attempts := service.NewAttemptService(r, mastery.DefaultConfig())

	// 造 6 个易错点：同一个知识点连续答错，会被判为 shaky。
	var kpIDs []int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy' AND EXISTS (SELECT 1 FROM questions q WHERE q.kp_id = kp.id)
		ORDER BY kp.id LIMIT 6`).Scan(&kpIDs).Error)
	require.Len(t, kpIDs, 6)

	for i, kpID := range kpIDs {
		for try := 0; try < 3; try++ {
			_, err := attempts.Report(1, []service.AttemptInput{{
				ClientID: string(rune('a'+i)) + string(rune('0'+try)),
				KpID:     kpID, IsCorrect: false, CostMs: 4000, Source: "quiz",
			}})
			require.NoError(t, err)
		}
	}

	detail, err := svc.Today(1)
	require.NoError(t, err)

	shaky := map[int64]bool{}
	for _, id := range kpIDs {
		shaky[id] = true
	}
	hit := 0
	for _, item := range detail.Items {
		if shaky[item.KpID] {
			hit++
			require.Equal(t, "shaky", item.Bucket)
		}
	}
	// 易错桶配额是 2，必须优先被占满。
	require.GreaterOrEqual(t, hit, 2, "易错点应该被优先排进计划")
}

func TestExtraPlanIsCappedPerDay(t *testing.T) {
	svc, _ := newPlanSvc(t)

	main, err := svc.Today(1)
	require.NoError(t, err)
	require.Equal(t, 1, main.Plan.SeqNo)

	second, err := svc.Extra(1)
	require.NoError(t, err)
	require.Equal(t, 2, second.Plan.SeqNo)
	require.NotEqual(t, main.Plan.ID, second.Plan.ID)

	third, err := svc.Extra(1)
	require.NoError(t, err)
	require.Equal(t, 3, third.Plan.SeqNo)

	_, err = svc.Extra(1)
	require.ErrorIs(t, err, service.ErrTooManyPlansToday, "一天最多 3 份，防止变成刷题")
}

func TestTodayReturnsLatestPlanSoRefreshKeepsExtra(t *testing.T) {
	svc, _ := newPlanSvc(t)

	main, err := svc.Today(1)
	require.NoError(t, err)
	extra, err := svc.Extra(1)
	require.NoError(t, err)
	require.NotEqual(t, main.Plan.ID, extra.Plan.ID)

	// 加餐后刷新页面不该被扔回已经做完的主计划。
	again, err := svc.Today(1)
	require.NoError(t, err)
	require.Equal(t, extra.Plan.ID, again.Plan.ID)
	require.Equal(t, 2, again.Plan.SeqNo)
}

func TestExtraPlanAvoidsRepeatingMainPlanQuestions(t *testing.T) {
	svc, _ := newPlanSvc(t)

	main, err := svc.Today(1)
	require.NoError(t, err)
	extra, err := svc.Extra(1)
	require.NoError(t, err)

	mainKps := map[int64]bool{}
	for _, item := range main.Items {
		mainKps[item.KpID] = true
	}
	overlap := 0
	for _, item := range extra.Items {
		if mainKps[item.KpID] {
			overlap++
		}
	}
	// 主计划已经练过的知识点在加餐里重复出现是浪费，允许少量重叠但不该整份相同。
	require.Less(t, overlap, len(extra.Items), "加餐不该和主计划完全重复")
}

func TestUnknownPlanReturnsNotFound(t *testing.T) {
	svc, _ := newPlanSvc(t)

	_, err := svc.Start(1, 99999)
	require.ErrorIs(t, err, service.ErrPlanNotFound)

	_, err = svc.Finish(1, 99999)
	require.ErrorIs(t, err, service.ErrPlanNotFound)
}

func TestPlanBelongingToAnotherChildIsNotAccessible(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)

	require.NoError(t, gdb.Create(&model.Child{ID: 2, Name: "另一个小朋友", Grade: "中班"}).Error)

	_, err = svc.Start(2, detail.Plan.ID)
	require.ErrorIs(t, err, service.ErrPlanNotFound, "不能碰别人的计划")

	_, err = svc.Answer(2, detail.Plan.ID, detail.Items[0].ID, service.AnswerInput{OptionIndex: 0})
	require.ErrorIs(t, err, service.ErrPlanNotFound)
}

func TestItemFromAnotherPlanIsRejected(t *testing.T) {
	svc, _ := newPlanSvc(t)

	main, err := svc.Today(1)
	require.NoError(t, err)
	extra, err := svc.Extra(1)
	require.NoError(t, err)

	_, err = svc.Answer(1, main.Plan.ID, extra.Items[0].ID, service.AnswerInput{OptionIndex: 0})
	require.ErrorIs(t, err, service.ErrPlanItemNotFound)
}

func TestHistoryListsPlansNewestFirst(t *testing.T) {
	svc, _ := newPlanSvc(t)

	_, err := svc.Today(1)
	require.NoError(t, err)
	_, err = svc.Extra(1)
	require.NoError(t, err)

	rows, err := svc.History(1, "", "", "")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, 1, rows[0].SeqNo)
	require.Equal(t, 2, rows[1].SeqNo)
	require.NotEmpty(t, rows[0].PlanDate)
	require.Len(t, rows[0].PlanDate, 10, "日期应该是 YYYY-MM-DD")

	totalSubjects := 0
	for _, s := range rows[0].Subjects {
		totalSubjects += s.Count
	}
	require.Equal(t, rows[0].TargetCount, totalSubjects, "学科分布数量之和应等于题量")
}

func TestReviewIncludesAnswerAndPicks(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	item := detail.Items[0]
	correct := correctIndexFor(t, gdb, item.Question.ID)
	wrong := (correct + 1) % 4

	_, err = svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: wrong})
	require.NoError(t, err)
	_, err = svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: correct})
	require.NoError(t, err)

	review, err := svc.Review(1, detail.Plan.ID)
	require.NoError(t, err)
	require.Len(t, review.Items, 10)

	got := review.Items[0]
	require.Equal(t, correct, got.Question.AnswerIndex)
	require.Equal(t, []int{wrong, correct}, got.Picks)
	require.Equal(t, "correct", got.Status)
	require.Equal(t, 2, got.Tries)
}

func TestReviewRejectsOtherChild(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)

	// 第二个孩子
	require.NoError(t, gdb.Exec(`INSERT INTO children (id, name, grade) VALUES (2, '测试', '')`).Error)

	_, err = svc.Review(2, detail.Plan.ID)
	require.ErrorIs(t, err, service.ErrPlanNotFound)
}

func TestHistoryStatusFilter(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	for _, item := range detail.Items {
		_, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{
			OptionIndex: correctIndexFor(t, gdb, item.Question.ID),
		})
		require.NoError(t, err)
	}
	_, err = svc.Finish(1, detail.Plan.ID)
	require.NoError(t, err)
	_, err = svc.Extra(1)
	require.NoError(t, err)

	done, err := svc.History(1, "", "", "done")
	require.NoError(t, err)
	require.Len(t, done, 1)
	require.Equal(t, "done", done[0].Status)

	pending, err := svc.History(1, "", "", "pending")
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, "pending", pending[0].Status)
}

func TestGenerateTodayIsIdempotent(t *testing.T) {
	svc, _ := newPlanSvc(t)

	first, err := svc.Today(1)
	require.NoError(t, err)
	second, err := svc.Today(1)
	require.NoError(t, err)
	require.Equal(t, first.Plan.ID, second.Plan.ID)
	require.Equal(t, 1, first.Plan.SeqNo)
}

func TestPlanWithoutQuestionBankFails(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	// 刻意不灌题库。

	r := repo.New(gdb)
	svc := service.NewPlanService(r, service.NewAttemptService(r, mastery.DefaultConfig()))

	_, err = svc.Today(1)
	require.ErrorIs(t, err, service.ErrNoCandidates)
}

func TestTodoExcludesDone(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	for _, item := range detail.Items {
		_, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{
			OptionIndex: correctIndexFor(t, gdb, item.Question.ID),
		})
		require.NoError(t, err)
	}
	_, err = svc.Finish(1, detail.Plan.ID)
	require.NoError(t, err)

	rows, err := svc.Todo(1)
	require.NoError(t, err)
	for _, r := range rows {
		require.NotEqual(t, detail.Plan.ID, r.ID)
	}
}

func TestTodoExcludesOutOfWindow(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	old := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	require.NoError(t, gdb.Create(&model.StudyPlan{
		ChildID: 1, PlanDate: old, SeqNo: 1,
		Status: "doing", TargetCount: 10, DoneCount: 3,
		CreatedAt: time.Now(),
	}).Error)

	rows, err := svc.Todo(1)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestTodoOrdersTodayFirst(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	require.NoError(t, gdb.Create(&model.StudyPlan{
		ChildID: 1, PlanDate: yesterday, SeqNo: 1,
		Status: "doing", TargetCount: 10, DoneCount: 2,
		CreatedAt: time.Now(),
	}).Error)

	today, err := svc.Today(1)
	require.NoError(t, err)

	rows, err := svc.Todo(1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2)
	require.Equal(t, today.Plan.ID, rows[0].ID, "今天的任务应排在补做前面")
}

func TestTodoIncludesSubjects(t *testing.T) {
	svc, _ := newPlanSvc(t)

	_, err := svc.Today(1)
	require.NoError(t, err)

	rows, err := svc.Todo(1)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	total := 0
	for _, s := range rows[0].Subjects {
		total += s.Count
	}
	require.Equal(t, rows[0].TargetCount, total)
}

func TestKidDetailRejectsOtherChild(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)

	require.NoError(t, gdb.Exec(`INSERT INTO children (id, name, grade) VALUES (2, '测试', '')`).Error)

	_, err = svc.Detail(2, detail.Plan.ID)
	require.ErrorIs(t, err, service.ErrPlanNotFound)
}

func TestKidDetailNeverExposesAnswer(t *testing.T) {
	svc, _ := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)

	got, err := svc.Detail(1, detail.Plan.ID)
	require.NoError(t, err)

	blob, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(blob), `"answer_index"`)
	require.NotContains(t, string(blob), `"answer"`)
}

func TestAnswerOnDonePlanDoesNotRescore(t *testing.T) {
	svc, gdb := newPlanSvc(t)

	detail, err := svc.Today(1)
	require.NoError(t, err)
	item := detail.Items[0]
	correct := correctIndexFor(t, gdb, item.Question.ID)

	for _, it := range detail.Items {
		_, err := svc.Answer(1, detail.Plan.ID, it.ID, service.AnswerInput{
			OptionIndex: correctIndexFor(t, gdb, it.Question.ID),
		})
		require.NoError(t, err)
	}
	fin, err := svc.Finish(1, detail.Plan.ID)
	require.NoError(t, err)
	correctBefore := fin.Plan.CorrectCount

	var attemptCount int64
	require.NoError(t, gdb.Raw(`SELECT COUNT(1) FROM attempts WHERE child_id = 1`).Scan(&attemptCount).Error)
	attemptsBefore := attemptCount

	res, err := svc.Answer(1, detail.Plan.ID, item.ID, service.AnswerInput{OptionIndex: correct})
	require.NoError(t, err)
	require.Equal(t, correctBefore, res.Plan.CorrectCount)

	require.NoError(t, gdb.Raw(`SELECT COUNT(1) FROM attempts WHERE child_id = 1`).Scan(&attemptCount).Error)
	require.Equal(t, attemptsBefore, attemptCount)
}
