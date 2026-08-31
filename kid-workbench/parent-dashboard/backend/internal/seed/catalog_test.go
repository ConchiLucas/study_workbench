package seed_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/seed"
)

func TestSeedCatalogCounts(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))

	var subjects, kps int64
	require.NoError(t, gdb.Model(&model.Subject{}).Count(&subjects).Error)
	require.NoError(t, gdb.Model(&model.KnowledgePoint{}).Count(&kps).Error)
	require.Equal(t, int64(10), subjects)
	require.Equal(t, int64(1236), kps)

	want := map[string]int64{
		"literacy": 300, "pinyin": 45, "math": 408, "english": 200,
		"science": 99, "poem": 50, "logic": 62, "chengyu": 32, "phrase": 32, "game": 8,
	}
	for code, n := range want {
		var got int64
		require.NoError(t, gdb.Raw(`
			SELECT COUNT(1) FROM knowledge_points kp
			JOIN modules m ON m.id = kp.module_id
			JOIN subjects s ON s.id = m.subject_id
			WHERE s.code = ?`, code).Scan(&got).Error)
		require.Equal(t, n, got, "学科 %s 知识点数不符", code)
	}
}

func TestSeedCatalogIsIdempotent(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, seed.Catalog(gdb))

	var kps int64
	require.NoError(t, gdb.Model(&model.KnowledgePoint{}).Count(&kps).Error)
	require.Equal(t, int64(1236), kps)
}

func TestDemoProducesAllStatuses(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, seed.Demo(gdb, mastery.DefaultConfig(), 1, 60))

	for _, status := range []string{"mastered", "learning", "shaky"} {
		var n int64
		require.NoError(t, gdb.Model(&model.MasteryState{}).
			Where("child_id = 1 AND status = ?", status).Count(&n).Error)
		require.Greater(t, n, int64(0), "缺少 %s 状态的示例数据", status)
	}

	var overdue int64
	require.NoError(t, gdb.Raw(`SELECT COUNT(1) FROM mastery_states
		WHERE child_id = 1 AND status = 'mastered' AND due_at < CURRENT_TIMESTAMP`).
		Scan(&overdue).Error)
	require.Greater(t, overdue, int64(0), "缺少 review_due 的示例数据")

	var days int64
	require.NoError(t, gdb.Model(&model.DailyStat{}).Where("child_id = 1").Count(&days).Error)
	require.Greater(t, days, int64(20), "日历热力图需要足够多的活跃天")

	// 每个学科都至少有一些练习记录
	type row struct {
		Code string
		N    int64
	}
	var rows []row
	require.NoError(t, gdb.Raw(`
		SELECT s.code, COUNT(1) AS n
		FROM mastery_states ms
		JOIN knowledge_points kp ON kp.id = ms.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE ms.child_id = 1
		GROUP BY s.code`).Scan(&rows).Error)
	require.GreaterOrEqual(t, len(rows), 8, "每个学科都应有学习数据")
	for _, r := range rows {
		require.Greater(t, r.N, int64(0), "学科 %s 没有掌握记录", r.Code)
	}
}

func TestRecomputeReproducesSameState(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, seed.Demo(gdb, mastery.DefaultConfig(), 1, 30))

	var before []model.MasteryState
	require.NoError(t, gdb.Where("child_id = 1").Order("kp_id").Find(&before).Error)

	require.NoError(t, seed.Recompute(gdb, mastery.DefaultConfig(), 1))

	var after []model.MasteryState
	require.NoError(t, gdb.Where("child_id = 1").Order("kp_id").Find(&after).Error)

	require.Len(t, after, len(before))
	for i := range before {
		require.Equal(t, before[i].KpID, after[i].KpID)
		require.Equal(t, before[i].Status, after[i].Status)
		require.Equal(t, before[i].Attempts, after[i].Attempts)
		require.Equal(t, before[i].Streak, after[i].Streak)
	}
}
