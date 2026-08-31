# Literacy KP Detail Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Literacy character detail drawer shows per-skill mastery cards (听/义/字) and answer history tagged by skill; character-level rollup status matches the matrix.

**Architecture:** Extend `DashboardService.KpDetail` to attach `skills[]` (from `mastery_skills` + `RollupSkills`) and `history[].skill_code` (join `attempts.question_id` → `questions.code` → `SkillFromQuestionCode`). Frontend `KpDetailDrawer` renders compact skill cards + pills only when `subject_code === 'literacy'`.

**Tech Stack:** Go (GORM), React + TypeScript, existing `mastery` package, Vite parent-dashboard. Docker for smoke (`make up`).

**Spec:** `docs/superpowers/specs/2026-08-29-literacy-kp-detail-skills-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `parent-dashboard/backend/internal/service/dashboard.go` | Extend `HistoryItem` / `KpDetail`; fill skills + skill_code in `KpDetail` |
| `parent-dashboard/backend/internal/service/service_test.go` | Tests for literacy skills on detail + history skill_code |
| `parent-dashboard/frontend/src/api/types.ts` | `skills?` on `KpDetail`, `skill_code?` on `HistoryItem` |
| `parent-dashboard/frontend/src/components/mastery/KpDetailDrawer.tsx` | Skill cards + history pills for literacy |

Reuse (do not reinvent): `mastery.LiteracySkills`, `mastery.SkillFromQuestionCode`, `mastery.RollupSkills`, `mastery.Display`, `skillToEngine` (same `service` package in `attempt.go`), `repo.ListMasterySkills`, FE `STATUS_STYLE` / matrix skill labels.

Skill codes (canonical): `listen_glyph` · `glyph_sense` · `sense_char`.

---

### Task 1: Backend — failing tests for KpDetail skills + history skill_code

**Files:**
- Modify: `parent-dashboard/backend/internal/service/service_test.go`
- Test: same file

- [ ] **Step 1: Add failing tests**

Append to `service_test.go` (keep existing `TestKpDetailIncludesHistory`):

```go
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
	require.Len(t, d.Skills, 3)
	require.Equal(t, mastery.LiteracySkills, []string{d.Skills[0].Code, d.Skills[1].Code, d.Skills[2].Code})
	for _, sk := range d.Skills {
		require.Contains(t, []string{"not_started", "learning", "shaky", "mastered", "review_due"}, sk.Status)
	}
}

func TestKpDetailHistorySkillCodeFromQuestion(t *testing.T) {
	svc, gdb := newDashboard(t)

	type pair struct{ KpID, QID int64 }
	var p pair
	require.NoError(t, gdb.Raw(`
		SELECT kp.id AS kp_id, q.id AS q_id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		JOIN questions q ON q.kp_id = kp.id
		WHERE s.code = 'literacy' AND q.code = ?
		ORDER BY kp.id LIMIT 1`, mastery.SkillGlyphSense).Scan(&p).Error)
	require.NotZero(t, p.KpID)
	require.NotZero(t, p.QID)

	qID := p.QID
	require.NoError(t, gdb.Create(&model.Attempt{
		ChildID: 1, KpID: p.KpID, QuestionID: &qID,
		IsCorrect: true, CostMs: 1200, Source: "quiz",
		ClientID: fmt.Sprintf("test-skill-%d", qID),
	}).Error)

	d, err := svc.KpDetail(1, p.KpID)
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
```

Add imports if missing: `fmt`, `github.com/conchi/study-workbench/internal/mastery`, `github.com/conchi/study-workbench/internal/model`.

- [ ] **Step 2: Run tests — expect compile/fail**

```bash
cd parent-dashboard/backend && go test ./internal/service/ -run 'TestKpDetailLiteracy|TestKpDetailHistorySkill|TestKpDetailParentMark' -count=1
```

Expected: compile error (`Skills` / `SkillCode` undefined) or FAIL until Task 2.

- [ ] **Step 3: Commit**

```bash
git add parent-dashboard/backend/internal/service/service_test.go
git commit -m "$(cat <<'EOF'
test: cover literacy skills and history skill_code on KpDetail

EOF
)"
```

---

### Task 2: Backend — extend KpDetail types and implementation

**Files:**
- Modify: `parent-dashboard/backend/internal/service/dashboard.go` (`HistoryItem`, `KpDetail`, `KpDetail` method)

- [ ] **Step 1: Extend structs**

In `dashboard.go`, change:

```go
type HistoryItem struct {
	At        time.Time `json:"at"`
	IsCorrect bool      `json:"is_correct"`
	CostMs    int       `json:"cost_ms"`
	Source    string    `json:"source"`
	SkillCode string    `json:"skill_code,omitempty"`
}

type KpDetail struct {
	KpID        int64         `json:"kp_id"`
	Title       string        `json:"title"`
	Payload     string        `json:"payload"`
	Difficulty  int           `json:"difficulty"`
	SubjectCode string        `json:"subject_code"`
	SubjectName string        `json:"subject_name"`
	ModuleName  string        `json:"module_name"`
	Status      string        `json:"status"`
	Attempts    int           `json:"attempts"`
	Correct     int           `json:"correct"`
	Accuracy    float64       `json:"accuracy"`
	Streak      int           `json:"streak"`
	BestStreak  int           `json:"best_streak"`
	DueAt       *time.Time    `json:"due_at"`
	MasteredAt  *time.Time    `json:"mastered_at"`
	Skills      []MatrixSkill `json:"skills,omitempty" gorm:"-"`
	History     []HistoryItem `json:"history" gorm:"-"`
}
```

Reuse existing `MatrixSkill` in the same file.

- [ ] **Step 2: Fill history with skill_code**

Replace the history query in `KpDetail` with a join that pulls question code, then map:

```go
type histRow struct {
	At           time.Time
	IsCorrect    bool
	CostMs       int
	Source       string
	QuestionCode *string
}
var hist []histRow
if err := s.repo.DB().Raw(`
	SELECT a.created_at AS at, a.is_correct, a.cost_ms, a.source, q.code AS question_code
	FROM attempts a
	LEFT JOIN questions q ON q.id = a.question_id
	WHERE a.child_id = ? AND a.kp_id = ?
	ORDER BY a.created_at ASC`, childID, kpID).Scan(&hist).Error; err != nil {
	return out, err
}
out.History = make([]HistoryItem, 0, len(hist))
for _, h := range hist {
	item := HistoryItem{At: h.At, IsCorrect: h.IsCorrect, CostMs: h.CostMs, Source: h.Source}
	if h.QuestionCode != nil {
		item.SkillCode = mastery.SkillFromQuestionCode(*h.QuestionCode)
	}
	out.History = append(out.History, item)
}
```

- [ ] **Step 3: Fill skills + rollup status for literacy**

After history load, before `return` — copy the same skill assembly pattern already used in `Matrix()`:

```go
if out.SubjectCode == "literacy" {
	now := time.Now()
	skillRows, err := s.repo.ListMasterySkills(s.repo.DB(), childID, []int64{kpID})
	if err != nil {
		return out, err
	}
	byCode := map[string]model.MasterySkill{}
	for _, sk := range skillRows {
		byCode[sk.SkillCode] = sk
	}
	statuses := make([]mastery.Status, 0, len(mastery.LiteracySkills))
	out.Skills = make([]MatrixSkill, 0, len(mastery.LiteracySkills))
	for _, code := range mastery.LiteracySkills {
		row, ok := byCode[code]
		st := mastery.StatusNotStarted
		attempts, correct := 0, 0
		if ok {
			eng := skillToEngine(row)
			st = mastery.Display(eng, now)
			attempts, correct = eng.Attempts, eng.Correct
		}
		acc := 0.0
		if attempts > 0 {
			acc = float64(correct) / float64(attempts)
		}
		out.Skills = append(out.Skills, MatrixSkill{
			Code: code, Status: string(st), Accuracy: acc, Attempts: attempts,
		})
		statuses = append(statuses, st)
	}
	out.Status = string(mastery.RollupSkills(statuses))
}
```

`skillToEngine` is already in `attempt.go` (same `service` package). Keep imports aligned with `Matrix()`.

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd parent-dashboard/backend && go test ./internal/service/ -run 'TestKpDetail' -count=1
```

Expected: PASS (including new + existing history test).

Also:

```bash
cd parent-dashboard/backend && go test ./internal/service/ ./internal/mastery/ -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add parent-dashboard/backend/internal/service/dashboard.go parent-dashboard/backend/internal/service/service_test.go
git commit -m "$(cat <<'EOF'
feat: return literacy skills and history skill_code on KpDetail

EOF
)"
```

---

### Task 3: Frontend types

**Files:**
- Modify: `parent-dashboard/frontend/src/api/types.ts`

- [ ] **Step 1: Extend types**

```ts
export interface HistoryItem {
  at: string; is_correct: boolean; cost_ms: number; source: string;
  skill_code?: string;
}

export interface KpDetail {
  kp_id: number; title: string; payload: string; difficulty: number;
  subject_code: string; subject_name: string; module_name: string;
  status: MasteryStatus; attempts: number; correct: number; accuracy: number;
  streak: number; best_streak: number;
  due_at: string | null; mastered_at: string | null;
  skills?: MatrixSkill[];
  history: HistoryItem[];
}
```

Reuse existing `MatrixSkill`.

- [ ] **Step 2: Commit**

```bash
git add parent-dashboard/frontend/src/api/types.ts
git commit -m "$(cat <<'EOF'
feat: type KpDetail skills and history skill_code

EOF
)"
```

---

### Task 4: Frontend — skill cards + history pills in KpDetailDrawer

**Files:**
- Modify: `parent-dashboard/frontend/src/components/mastery/KpDetailDrawer.tsx`
- Reference mockup: `docs/superpowers/mockups/2026-08-29-literacy-kp-detail.html`

- [ ] **Step 1: Add skill label helpers at top of file**

```tsx
const SKILL_SHORT: Record<string, string> = {
  listen_glyph: '听',
  glyph_sense: '义',
  sense_char: '字',
}
const SKILL_FULL: Record<string, string> = {
  listen_glyph: '听音选字',
  glyph_sense: '看字选义',
  sense_char: '看义选字',
}
```

Import `MatrixSkill` from `../../api/types` if needed.

- [ ] **Step 2: Insert skill section after status banner, before the six-stat grid**

Only when literacy:

```tsx
{data.subject_code === 'literacy' && (
  <div>
    <div className="mb-2 text-sm font-medium text-slate-600">题型掌握</div>
    <div className="space-y-2">
      {(data.skills ?? [
        { code: 'listen_glyph', status: 'not_started' as const, accuracy: 0, attempts: 0 },
        { code: 'glyph_sense', status: 'not_started' as const, accuracy: 0, attempts: 0 },
        { code: 'sense_char', status: 'not_started' as const, accuracy: 0, attempts: 0 },
      ]).map((sk) => {
        const st = STATUS_STYLE[sk.status]
        return (
          <div
            key={sk.code}
            className="flex items-center gap-3 rounded-xl border border-slate-100 bg-slate-50 px-3 py-2"
          >
            <span
              className="grid h-9 w-9 place-items-center rounded-lg text-sm font-bold text-white"
              style={{ backgroundColor: st.ring }}
            >
              {SKILL_SHORT[sk.code] ?? sk.code}
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium text-slate-700">
                {SKILL_FULL[sk.code] ?? sk.code}
              </div>
              <div className="text-[11px] text-slate-400">
                练 {sk.attempts} 次 · 正确率 {Math.round(sk.accuracy * 100)}%
              </div>
            </div>
            <span
              className="shrink-0 rounded-full px-2 py-0.5 text-[11px] font-semibold"
              style={{ backgroundColor: st.bg, color: st.text }}
            >
              {st.label}
            </span>
          </div>
        )
      })}
    </div>
  </div>
)}
```

If `STATUS_STYLE` entries use different property names in this file (e.g. already `style!.bg`), keep the same property names as the status banner above.

Add a small heading above the existing six-stat grid when literacy, e.g. `字级汇总`, optional but matches mockup.

- [ ] **Step 3: Tag history rows**

Replace the history list item left side so skill / mark pills show:

```tsx
<li key={i} className="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2">
  <span className="flex items-center gap-2 text-slate-500">
    {h.source === 'parent_mark' ? (
      <span className="rounded bg-brand-500 px-1.5 py-0.5 text-[10px] font-bold text-white">标</span>
    ) : h.skill_code ? (
      <span className="rounded bg-teal-600 px-1.5 py-0.5 text-[10px] font-bold text-white">
        {SKILL_SHORT[h.skill_code] ?? h.skill_code}
      </span>
    ) : null}
    <span>{h.at.slice(0, 16).replace('T', ' ')}</span>
  </span>
  <span className="flex items-center gap-2">
    {h.source === 'parent_mark'
      ? <em className="text-brand-700">家长标记</em>
      : <span>{(h.cost_ms / 1000).toFixed(1)}s</span>}
    <span>{h.is_correct ? '✅' : '❌'}</span>
  </span>
</li>
```

Keep existing class tokens in this file (`bg-brand-500`, `text-brand-700`, `rounded-xl2`).

- [ ] **Step 4: Typecheck / build**

```bash
cd parent-dashboard/frontend && npm run build
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git add parent-dashboard/frontend/src/components/mastery/KpDetailDrawer.tsx parent-dashboard/frontend/src/api/types.ts
git commit -m "$(cat <<'EOF'
feat: show literacy skill cards and tagged history in KP detail

EOF
)"
```

---

### Task 5: Docker rebuild + smoke

**Files:** none (ops)

- [ ] **Step 1: Rebuild stack**

```bash
cd /Users/conchi/workforce/go_workforce/study_workbench && make up
```

Expected: backend healthy on `:19081`.

- [ ] **Step 2: API smoke**

```bash
KP=$(curl -sS 'http://localhost:19081/api/v1/children/1/mastery/matrix?subject=literacy' | python3 -c 'import sys,json;d=json.load(sys.stdin);print((d.get("data") or d)["modules"][0]["points"][0]["id"])')
curl -sS "http://localhost:19081/api/v1/children/1/knowledge-points/${KP}" | python3 -c '
import sys,json
d=json.load(sys.stdin)
data=d.get("data") or d
assert data["subject_code"]=="literacy"
assert len(data["skills"])==3
assert [s["code"] for s in data["skills"]]==["listen_glyph","glyph_sense","sense_char"]
print("skills ok", data["skills"][0])
print("history sample", (data.get("history") or [])[:2])
'
```

Expected: assertions pass.

- [ ] **Step 3: UI smoke**

Open `http://localhost:19081/subjects/literacy`, click a character. Confirm:
1. 「题型掌握」three cards  
2. Six-stat grid still present  
3. History rows can show 听/义/字/标 pills when data exists  

- [ ] **Step 4: Final commit if smoke needed tiny fixes** (only if you changed files)

---

## Self-review checklist

1. **Spec coverage:** skills cards (A) + tagged history (C) + rollup status + no schema change + literacy-only UI — all tasked.  
2. **No placeholders:** concrete code, commands, paths.  
3. **Type consistency:** `listen_glyph` / `glyph_sense` / `sense_char`; JSON `skill_code`, `skills`, `MatrixSkill` shape matches matrix.
