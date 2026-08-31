package math_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/math"
)

func setupMathDB(t *testing.T) *math.Service {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'math', '算术', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'add10', '20以内加法', 0),
 (2, 1, 'sub10', '20以内减法', 1),
 (3, 1, 'shape', '认识图形', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (1, 1, '1p2', '1+2', '{"kind":"add","a":1,"b":2}', 1, 0),
 (2, 2, '5m3', '5-3', '{"kind":"sub","a":5,"b":3}', 2, 0),
 (3, 3, 's1', '圆形', '{}', 1, 0),
 (99, 1, 'gone', '旧题', '{"kind":"add","a":1,"b":1}', 1, 9);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return math.NewService(gdb, nil, nil, nil, nil)
}

func TestSyncAndListGroups(t *testing.T) {
	svc := setupMathDB(t)
	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 4, res.Total)
	require.Equal(t, 4, res.Upserted)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 4, list.Total)
	require.Len(t, list.Groups, 3)
	require.Equal(t, "add10", list.Groups[0].ModuleCode)
	require.Equal(t, "1+2", list.Groups[0].Items[0].Title)
	require.Equal(t, "add", list.Groups[0].Items[0].Kind)
	require.Equal(t, "sub", list.Groups[1].Items[0].Kind)
	require.Equal(t, "圆形", list.Groups[2].Items[0].Title)
}

func TestSyncPrunesRemovedKPs(t *testing.T) {
	svc := setupMathDB(t)
	_, err := svc.Sync()
	require.NoError(t, err)

	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`DELETE FROM knowledge_points WHERE id = 99`).Error)

	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 3, res.Total)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 3, list.Total)
	for _, g := range list.Groups {
		for _, it := range g.Items {
			require.NotEqual(t, int64(99), it.KpID)
		}
	}
}

func TestSyncSetsSpeechText(t *testing.T) {
	svc := setupMathDB(t)
	_, err := svc.Sync()
	require.NoError(t, err)
	list, err := svc.List("groups")
	require.NoError(t, err)
	byTitle := map[string]string{}
	for _, g := range list.Groups {
		for _, it := range g.Items {
			byTitle[it.Title] = it.SpeechText
		}
	}
	require.Equal(t, "一加二", byTitle["1+2"])
	require.Equal(t, "五减三", byTitle["5-3"])
	require.Equal(t, "圆形", byTitle["圆形"])
}

func TestBatchGenerateSpeechRequiresModule(t *testing.T) {
	svc := setupMathDB(t)
	_, err := svc.BatchGenerateSpeech(context.Background(), "")
	require.Error(t, err)
}
