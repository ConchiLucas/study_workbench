package qtask_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/qtask"
)

func setup(t *testing.T) *qtask.Service {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, order_no INT);
CREATE TABLE questions (id INTEGER PRIMARY KEY, kp_id INT, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, speech TEXT, difficulty INT);
INSERT INTO subjects(id, code) VALUES (1, 'literacy');
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'g1', '第1组', 1);
INSERT INTO knowledge_points(id, module_id, code, title, order_no) VALUES
 (1,1,'c1','一',1),(2,1,'c2','二',2),(3,1,'c3','三',3),(4,1,'c4','四',4),(5,1,'c5','五',5),
 (6,1,'c6','六',6),(7,1,'c7','七',7),(8,1,'c8','八',8),(9,1,'c9','九',9),(10,1,'c10','十',10);
INSERT INTO questions(id, kp_id, code, type, stem, options, answer, visual, speech, difficulty) VALUES
 (101,1,'glyph_sense','choice','看字图，选出义图','[{"label":"一"},{"label":"二"},{"label":"三"},{"label":"四"}]','{"index":0}','','{"text":"一","lang":"zh-CN"}',1),
 (102,1,'sense_char','choice','看义图，选出字','[{"label":"一"},{"label":"二"},{"label":"三"},{"label":"五"}]','{"index":0}','','{"text":"一","lang":"zh-CN"}',1),
 (103,2,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (104,3,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (105,4,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (106,5,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (107,6,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (108,7,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (109,8,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (110,9,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (111,10,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (112,2,'listen_glyph','choice','听一听，点出这个字','[]','{"index":0}','','{}',1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return qtask.NewService(gdb)
}

func TestCreateLiteracyTaskPicksTenUniqueQuestions(t *testing.T) {
	svc := setup(t)
	task, err := svc.Create(qtask.CreateInput{
		SubjectCode: "literacy",
		ModuleCode:  "g1",
	})
	require.NoError(t, err)
	require.Equal(t, "draft", task.Status)
	require.Equal(t, 10, task.TargetCount)
	require.Equal(t, "第1组", task.ModuleName)
	require.Contains(t, task.Title, "第1组")
	require.Len(t, task.Items, 10)

	seen := map[int64]struct{}{}
	for i, it := range task.Items {
		require.Equal(t, i+1, it.Seq)
		_, dup := seen[it.QuestionID]
		require.False(t, dup)
		seen[it.QuestionID] = struct{}{}
		require.NotEmpty(t, it.Stem)
		require.NotEmpty(t, it.CharText)
	}
}

func TestCreateFailsWhenFewerThanTenQuestions(t *testing.T) {
	svc := setup(t)
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`DELETE FROM questions WHERE id > 105`).Error)
	_, err = svc.Create(qtask.CreateInput{SubjectCode: "literacy", ModuleCode: "g1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "可出题")
}

func TestPublishBlocksReshuffle(t *testing.T) {
	svc := setup(t)
	task, err := svc.Create(qtask.CreateInput{SubjectCode: "literacy", ModuleCode: "g1"})
	require.NoError(t, err)
	_, err = svc.Publish(task.ID)
	require.NoError(t, err)
	_, err = svc.Reshuffle(task.ID)
	require.Error(t, err)
}

func TestUnpublishThenReshuffle(t *testing.T) {
	svc := setup(t)
	task, err := svc.Create(qtask.CreateInput{SubjectCode: "literacy", ModuleCode: "g1"})
	require.NoError(t, err)
	_, err = svc.Publish(task.ID)
	require.NoError(t, err)
	_, err = svc.Unpublish(task.ID)
	require.NoError(t, err)
	again, err := svc.Reshuffle(task.ID)
	require.NoError(t, err)
	require.Len(t, again.Items, 10)
}

func TestDeleteDraftOnly(t *testing.T) {
	svc := setup(t)
	task, err := svc.Create(qtask.CreateInput{SubjectCode: "literacy", ModuleCode: "g1"})
	require.NoError(t, err)
	_, err = svc.Publish(task.ID)
	require.NoError(t, err)
	require.Error(t, svc.Delete(task.ID))
	_, err = svc.Unpublish(task.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Delete(task.ID))
}

func TestListFiltersByStatus(t *testing.T) {
	svc := setup(t)
	a, err := svc.Create(qtask.CreateInput{SubjectCode: "literacy", ModuleCode: "g1", Title: "A"})
	require.NoError(t, err)
	b, err := svc.Create(qtask.CreateInput{SubjectCode: "literacy", ModuleCode: "g1", Title: "B"})
	require.NoError(t, err)
	_, err = svc.Publish(b.ID)
	require.NoError(t, err)
	drafts, err := svc.List("literacy", "draft")
	require.NoError(t, err)
	require.True(t, len(drafts) >= 1)
	for _, d := range drafts {
		require.Equal(t, "draft", d.Status)
	}
	_ = a
}
