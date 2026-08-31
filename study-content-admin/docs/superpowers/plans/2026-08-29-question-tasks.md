# Question Tasks (识字题包) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In study-content-admin, add a「题目管理」menu that creates literacy question packs (10 unique questions from one literacy group), with draft/publish workflow and preview — admin only, no kid-app.

**Architecture:** New package `internal/qtask` reads shared `modules` / `knowledge_points` / `questions` and writes `question_tasks` + `question_task_items`. Gin handlers under `/api/v1/question-tasks`. Frontend page at `/questions` mirrors literacy/math admin list+detail patterns.

**Tech Stack:** Go 1.26, GORM, Gin, Postgres (shared `study_workbench`), React + React Query + Vite (content-admin frontend)

**Spec:** `docs/superpowers/specs/2026-08-29-question-tasks-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `backend/internal/qtask/service.go` | Models, pick 10 questions, CRUD + publish |
| `backend/internal/qtask/service_test.go` | Unit tests with in-memory SQLite |
| `backend/internal/db/db.go` | `Migrate` creates `question_tasks` / `question_task_items` |
| `backend/internal/http/handler_qtask.go` | HTTP handlers |
| `backend/internal/http/router.go` | Wire routes + `Deps.QTask` |
| `backend/cmd/server/main.go` | Construct `qtask.NewService(gdb)` |
| `frontend/src/api/qtaskTypes.ts` | DTOs |
| `frontend/src/api/qtask.ts` | fetch helpers |
| `frontend/src/features/qtask/QuestionsPage.tsx` | List + create + detail UI |
| `frontend/src/App.tsx` | Route `/questions` |
| `frontend/src/layout/AppShell.tsx` | Nav「题目管理」 |
| `frontend/src/styles/global.css` | Minimal list/detail styles if needed |

---

### Task 1: Migrate tables

**Files:**
- Modify: `backend/internal/db/db.go`
- Test: exercise via Task 2 setup (Migrate in tests)

- [ ] **Step 1: Append migration SQL at end of `Migrate` (before final `return nil`)**

Postgres + SQLite variants (follow existing `math_assets` pattern in the same file):

```sql
-- postgres
CREATE TABLE IF NOT EXISTS question_tasks (
  id BIGSERIAL PRIMARY KEY,
  subject_code VARCHAR(30) NOT NULL,
  title VARCHAR(80) NOT NULL,
  module_code VARCHAR(50) NOT NULL,
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  target_count INT NOT NULL DEFAULT 10,
  status VARCHAR(20) NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_question_tasks_subject_status ON question_tasks(subject_code, status);
CREATE INDEX IF NOT EXISTS idx_question_tasks_module ON question_tasks(module_code);

CREATE TABLE IF NOT EXISTS question_task_items (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES question_tasks(id) ON DELETE CASCADE,
  seq INT NOT NULL,
  kp_id BIGINT NOT NULL,
  question_id BIGINT NOT NULL,
  UNIQUE(task_id, seq),
  UNIQUE(task_id, question_id)
);
```

SQLite: use `INTEGER PRIMARY KEY AUTOINCREMENT`, `DATETIME`, and `REFERENCES question_tasks(id) ON DELETE CASCADE` (PRAGMA foreign_keys already ON).

- [ ] **Step 2: Commit**

```bash
cd /Users/conchi/workforce/go_workforce/study-content-admin
git add backend/internal/db/db.go
git commit -m "db: add question_tasks and question_task_items tables"
```

---

### Task 2: qtask service — create pack (TDD)

**Files:**
- Create: `backend/internal/qtask/service.go`
- Create: `backend/internal/qtask/service_test.go`

- [ ] **Step 1: Write failing tests**

```go
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
 (101,1,'listen1','choice','听一听，点出这个字','[{"label":"一"},{"label":"二"},{"label":"三"},{"label":"四"}]','{"index":0}','','{"text":"一","lang":"zh-CN"}',1),
 (102,1,'listen2','choice','听一听，点出这个字','[{"label":"一"},{"label":"二"},{"label":"三"},{"label":"五"}]','{"index":0}','','{"text":"一","lang":"zh-CN"}',1),
 (103,2,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (104,3,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (105,4,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (106,5,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (107,6,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (108,7,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (109,8,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (110,9,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1),
 (111,10,'listen1','choice','听一听，点出这个字','[]','{"index":0}','','{}',1);
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
```

- [ ] **Step 2: Run tests — expect FAIL (package missing)**

```bash
cd /Users/conchi/workforce/go_workforce/study-content-admin/backend
go test ./internal/qtask/ -count=1
```

Expected: fail to compile / package not found

- [ ] **Step 3: Implement `service.go` (minimal)**

Key types and methods:

```go
package qtask

const TargetCount = 10
const StatusDraft = "draft"
const StatusPublished = "published"

type CreateInput struct {
	SubjectCode string
	ModuleCode  string
	Title       string // optional
}

type TaskDTO struct {
	ID          int64     `json:"id"`
	SubjectCode string    `json:"subjectCode"`
	Title       string    `json:"title"`
	ModuleCode  string    `json:"moduleCode"`
	ModuleName  string    `json:"moduleName"`
	TargetCount int       `json:"targetCount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Items       []ItemDTO `json:"items,omitempty"`
}

type ItemDTO struct {
	Seq         int             `json:"seq"`
	KpID        int64           `json:"kpId"`
	QuestionID  int64           `json:"questionId"`
	CharText    string          `json:"charText"`
	Code        string          `json:"code"`
	Stem        string          `json:"stem"`
	Options     json.RawMessage `json:"options"`
	AnswerIndex int             `json:"answerIndex"`
	Speech      json.RawMessage `json:"speech"`
}

type Service struct{ db *gorm.DB }
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Create(in CreateInput) (TaskDTO, error)
```

`Create` logic:
1. Require `subjectCode == "literacy"` (v1).
2. Load module: `JOIN subjects WHERE s.code=? AND m.code=?`.
3. Candidate query:

```sql
SELECT q.id AS question_id, q.kp_id, kp.title AS char_text, q.code, q.stem, q.options, q.answer, q.speech
FROM questions q
JOIN knowledge_points kp ON kp.id = q.kp_id
JOIN modules m ON m.id = kp.module_id
JOIN subjects s ON s.id = m.subject_id
WHERE s.code = ? AND m.code = ?
ORDER BY q.id
```

4. If `len(candidates) < TargetCount` → `fmt.Errorf("本组可出题仅 %d 道，需要 %d 道", n, TargetCount)`.
5. Shuffle with `rand.New(rand.NewSource(time.Now().UnixNano()))`, take 10.
6. Transaction: insert `question_tasks` (title default `识字 · {moduleName}`), insert items seq 1..10.
7. Return `Get(id)`.

Parse answer: `json.Unmarshal(answer, &struct{ Index int \`json:"index"\` })`.

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/qtask/ -count=1 -v
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/qtask/
git commit -m "feat(qtask): create literacy task with 10 unique questions"
```

---

### Task 3: List, Get, Reshuffle, Publish, Unpublish, Delete

**Files:**
- Modify: `backend/internal/qtask/service.go`
- Modify: `backend/internal/qtask/service_test.go`

- [ ] **Step 1: Add failing tests**

```go
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
```

- [ ] **Step 2: Run — expect FAIL (methods missing)**

```bash
go test ./internal/qtask/ -count=1
```

- [ ] **Step 3: Implement methods**

```go
func (s *Service) List(subject, status string) ([]TaskDTO, error) // no items
func (s *Service) Get(id int64) (TaskDTO, error)                 // with items
func (s *Service) Reshuffle(id int64) (TaskDTO, error)           // draft only; delete items; re-pick
func (s *Service) Publish(id int64) (TaskDTO, error)
func (s *Service) Unpublish(id int64) (TaskDTO, error)
func (s *Service) Delete(id int64) error                         // draft only
func (s *Service) ListLiteracyModules() ([]ModuleOption, error)  // for create form
```

`ModuleOption`: `{ Code string, Name string, Order int }`.

`Reshuffle`: load task; if not draft → error; same candidate query as Create; replace items in transaction.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/qtask/ -count=1 -v
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/qtask/
git commit -m "feat(qtask): list get publish reshuffle delete"
```

---

### Task 4: HTTP handlers + router

**Files:**
- Create: `backend/internal/http/handler_qtask.go`
- Modify: `backend/internal/http/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add `QTask *qtask.Service` to `Deps` and routes**

```go
v1.GET("/question-tasks", h.listQuestionTasks)
v1.POST("/question-tasks", h.createQuestionTask)
v1.GET("/question-tasks/literacy-modules", h.listQTaskLiteracyModules) // BEFORE :id
v1.GET("/question-tasks/:id", h.getQuestionTask)
v1.POST("/question-tasks/:id/reshuffle", h.reshuffleQuestionTask)
v1.POST("/question-tasks/:id/publish", h.publishQuestionTask)
v1.POST("/question-tasks/:id/unpublish", h.unpublishQuestionTask)
v1.DELETE("/question-tasks/:id", h.deleteQuestionTask)
```

Register `/literacy-modules` **before** `/:id` so Gin does not capture it as id.

- [ ] **Step 2: Implement handlers** (mirror `handler_math.go` style)

Create body:

```go
var body struct {
	SubjectCode string `json:"subjectCode"`
	ModuleCode  string `json:"moduleCode"`
	Title       string `json:"title"`
}
```

Map service errors containing `可出题` / `draft` / `published` → 400; `gorm.ErrRecordNotFound` → 404.

- [ ] **Step 3: Wire main.go**

```go
QTask: qtask.NewService(gdb),
```

- [ ] **Step 4: Compile**

```bash
cd backend && go build -o /dev/null ./cmd/server/ && go test ./internal/qtask/ ./internal/http/ -count=1
```

Expected: PASS (http package may only have catalog tests)

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/handler_qtask.go backend/internal/http/router.go backend/cmd/server/main.go
git commit -m "feat(qtask): expose question-tasks HTTP API"
```

---

### Task 5: Frontend API + Types

**Files:**
- Create: `frontend/src/api/qtaskTypes.ts`
- Create: `frontend/src/api/qtask.ts`

- [ ] **Step 1: Types**

```ts
export interface QTaskItem {
  seq: number
  kpId: number
  questionId: number
  charText: string
  code: string
  stem: string
  options: { label?: string }[]
  answerIndex: number
  speech?: { text?: string; lang?: string }
}

export interface QuestionTask {
  id: number
  subjectCode: string
  title: string
  moduleCode: string
  moduleName: string
  targetCount: number
  status: 'draft' | 'published'
  createdAt: string
  updatedAt: string
  items?: QTaskItem[]
}

export interface LiteracyModuleOption {
  code: string
  name: string
  order: number
}
```

- [ ] **Step 2: API helpers** (same `parseJSON` pattern as `math.ts`)

```ts
listQuestionTasks({ subject?, status? })
createQuestionTask({ subjectCode, moduleCode, title? })
getQuestionTask(id)
reshuffleQuestionTask(id)
publishQuestionTask(id)
unpublishQuestionTask(id)
deleteQuestionTask(id)
listLiteracyModules()
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/qtaskTypes.ts frontend/src/api/qtask.ts
git commit -m "feat(qtask): frontend API client"
```

---

### Task 6: QuestionsPage UI + nav

**Files:**
- Create: `frontend/src/features/qtask/QuestionsPage.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/layout/AppShell.tsx`
- Modify: `frontend/src/styles/global.css` (only if needed)

- [ ] **Step 1: Add NavLink + route**

AppShell (with literacy/math/english peers):

```tsx
<NavLink to="/questions" className={...}>题目管理</NavLink>
```

App.tsx:

```tsx
<Route path="/questions" element={<QuestionsPage />} />
```

- [ ] **Step 2: Implement `QuestionsPage`**

State: `selectedId: number | null` (null = list; set = detail). Or use query `?id=`.

**List view**
- Heading: `CONTENT / QUESTIONS` / 「题目管理」
- Button「新建识字任务」→ create panel: select module from `listLiteracyModules()`, optional title, submit `createQuestionTask` → open detail
- Table/cards: title, moduleName, status badge, updatedAt
- Actions: 查看 → setSelectedId; draft: 发布 / 删除; published: 撤回

**Detail view**
- Back to list
- Show title, status, module
- draft: 「重新抽题」「发布」
- published: 「撤回草稿」
- Ordered list of 10 items: seq, charText, code, stem, options with correct highlighted (`answerIndex`), play speech if `speech?.text` via `speechSynthesis` **or** skip play if no asset URL (v1: show speech text only is OK)

Reuse classes: `literacy-page`, `page-heading`, `refresh-button`, `error-panel`, `info-panel`, `mini-btn`, `group-list`.

- [ ] **Step 3: Manual smoke (Docker)**

```bash
cd /Users/conchi/workforce/go_workforce/study-content-admin && make up
# ensure literacy questions exist in shared DB (workbench seed)
curl -sS -X POST http://localhost:19091/api/v1/question-tasks \
  -H 'Content-Type: application/json' \
  -d '{"subjectCode":"literacy","moduleCode":"g1"}'
curl -sS 'http://localhost:19091/api/v1/question-tasks?subject=literacy'
```

Open `http://localhost:19091/questions` — create, preview 10, publish, unpublish.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/features/qtask/ frontend/src/App.tsx frontend/src/layout/AppShell.tsx frontend/src/styles/global.css
git commit -m "feat(qtask): questions management admin UI"
```

---

### Task 7: Acceptance checklist

- [ ] **Step 1: Verify against spec**

| Spec criterion | Check |
|---|---|
| 顶栏题目管理 | Nav + `/questions` |
| 按组生成 draft 且 10 题 | Create API + UI |
| 无重复 question_id | Test + UNIQUE |
| &lt;10 失败提示 N | TestCreateFails… |
| draft 重抽/删；published 只读可撤回 | Tests + UI |
| 详情展示 stem/options/正确项 | Detail UI |
| 不改 study_plans / kid-app | No files touched there |

- [ ] **Step 2: Final commit if any fixups**

```bash
git status
# commit only if needed
```

---

## Spec coverage (self-review)

| Spec section | Task |
|---|---|
| Tables | 1 |
| Create + pick 10 + unique + insufficient error | 2 |
| Reshuffle / publish / unpublish / delete / list | 3 |
| REST API | 4 |
| Admin UI + nav | 5–6 |
| Non-goals (no kid-app, no other subjects) | respected — only literacy Create gate |

No TBD placeholders. DTO field names consistent across Go JSON tags and TS interfaces (`subjectCode`, `moduleCode`, `answerIndex`, etc.).
