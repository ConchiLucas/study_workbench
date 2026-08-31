package pinyin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/pinyin"
)

func setupPinyinDB(t *testing.T) *pinyin.Service {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'pinyin', '拼音', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'shengmu', '声母', 1),
 (2, 1, 'yunmu', '韵母', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (101, 1, 'b', 'b', '{}', 1, 1),
 (102, 1, 'p', 'p', '{}', 1, 2),
 (201, 2, 'a', 'a', '{}', 1, 1),
 (202, 2, 'eng', 'eng', '{}', 1, 2);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return pinyin.NewService(gdb, nil, nil, nil, nil)
}

func TestSyncAndListGroups(t *testing.T) {
	svc := setupPinyinDB(t)
	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 4, res.Total)
	require.Equal(t, 4, res.Upserted)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 4, list.Total)
	require.Len(t, list.Groups, 2)
	require.Equal(t, "shengmu", list.Groups[0].ModuleCode)
	require.Equal(t, "yunmu", list.Groups[1].ModuleCode)

	byLetter := map[string]pinyin.ItemDTO{}
	for _, g := range list.Groups {
		for _, it := range g.Items {
			byLetter[it.Letter] = it
		}
	}
	require.Equal(t, "波", byLetter["b"].SoloText)
	require.Equal(t, "爸", byLetter["b"].WordText)
	require.Equal(t, "", byLetter["eng"].SoloText)
	require.Equal(t, "灯", byLetter["eng"].WordText)
}

func TestBatchGenerateSpeechRequiresModule(t *testing.T) {
	svc := setupPinyinDB(t)
	_, err := svc.BatchGenerateSpeech(context.Background(), "")
	require.Error(t, err)
}

func TestBatchGenerateGlyphsRequiresModule(t *testing.T) {
	svc := setupPinyinDB(t)
	_, err := svc.BatchGenerateGlyphs(context.Background(), "")
	require.Error(t, err)
}
