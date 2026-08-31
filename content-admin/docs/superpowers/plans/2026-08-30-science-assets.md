# Science (科普) Assets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a 科普 (`science`) content-admin module mirroring English: sync KPs from workbench, browse groups/table, generate glyph + sense + speech; every item defaults to needing a sense image (overridable).

**Architecture:** Copy the `english` package/API/UI shape into `science`, pointing sync at `subjects.code = 'science'`. Reuse glyph `RenderEnglishPNG` for multi-rune Chinese titles on a white card, add `sense.PromptScience`, MinIO keys under `science/…`, and wire nav `/science`.

**Tech Stack:** Go (Gin/GORM), React+TS (Vite), MinIO, shared-config image/TTS providers (Grok-first for sense).

**Spec:** `docs/superpowers/specs/2026-08-30-science-assets-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `backend/internal/science/rules.go` | `NeedsSenseImage` always `true` |
| `backend/internal/science/rules_test.go` | Rule tests |
| `backend/internal/science/service.go` | Asset model, Sync/List/Patch/Generate*/Batch* |
| `backend/internal/db/db.go` | `science_assets` DDL (pg + sqlite) |
| `backend/internal/storage/minio.go` | `ScienceGlyph/Sense/Speech` object keys |
| `backend/internal/sense/science.go` (+ test) | `PromptScience(title)` |
| `backend/internal/http/handler_science.go` | HTTP handlers |
| `backend/internal/http/router.go` | `/api/v1/science/...` routes + Deps |
| `backend/cmd/server/main.go` | Construct `science.Service` |
| `frontend/src/api/scienceTypes.ts` | DTOs |
| `frontend/src/api/science.ts` | fetch helpers |
| `frontend/src/features/science/SciencePage.tsx` | Page UI (clone EnglishPage) |
| `frontend/src/App.tsx` / `layout/AppShell.tsx` | Route + nav |

Mirror field names from english: `needs_sense_image`, `needs_sense_image_override`, `kp_order`, `synced_at`. Text column: `title` (not `word_text`).

---

### Task 1: Rules — always needs sense image

**Files:**
- Create: `backend/internal/science/rules.go`
- Create: `backend/internal/science/rules_test.go`

- [ ] **Step 1: Write failing test**

```go
package science_test

import (
	"testing"

	"github.com/conchi/study-content-admin/internal/science"
	"github.com/stretchr/testify/assert"
)

func TestNeedsSenseImageAlwaysTrue(t *testing.T) {
	assert.True(t, science.NeedsSenseImage("冬眠"))
	assert.True(t, science.NeedsSenseImage("光合作用"))
	assert.True(t, science.NeedsSenseImage(""))
}
```

- [ ] **Step 2: Run — expect fail**

```bash
cd backend && go test ./internal/science/ -run TestNeedsSenseImageAlwaysTrue -count=1
```

Expected: package not found / undefined.

- [ ] **Step 3: Implement**

```go
package science

// NeedsSenseImage is always true for science concepts (spec: 每种都要义图).
func NeedsSenseImage(_ string) bool { return true }
```

- [ ] **Step 4: Run — expect pass**

```bash
cd backend && go test ./internal/science/ -run TestNeedsSenseImageAlwaysTrue -count=1
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/science/rules.go backend/internal/science/rules_test.go
git commit -m "$(cat <<'EOF'
feat(science): always require sense images by rule

EOF
)"
```

---

### Task 2: PromptScience

**Files:**
- Create: `backend/internal/sense/science.go`
- Modify: `backend/internal/sense/prompt_test.go` (or create `science_test.go`)

- [ ] **Step 1: Failing test**

```go
func TestPromptScienceNoHanInOutputWhenMapped(t *testing.T) {
	p := sense.PromptScience("冬眠")
	require.NotEmpty(t, p)
	require.Contains(t, p, "child-friendly")
	// Prefer English subject phrase; if fallback uses title, ContainsHan may be true — assert sticker style suffix present.
	require.Contains(t, p, "sticker") // or whatever styleSuffix substring exists in prompt.go
}
```

Inspect `styleSuffix` in `sense/prompt.go` and assert a stable substring.

- [ ] **Step 2: Run — fail**

```bash
cd backend && go test ./internal/sense/ -run TestPromptScience -count=1
```

- [ ] **Step 3: Implement `PromptScience`**

Mirror `PromptEnglish`: build English-only subject for known titles where possible; fallback:

```go
fmt.Sprintf("a single simple child-friendly cartoon scene that clearly shows the science concept %q for young children, with no writing on it", title)
```

Then wrap with same style suffix as literacy/english. If `ContainsHan(p)`, replace with Han-free fallback that still names the concept in quotes only if unavoidable — prefer a small `scienceSubjects` map for the 99 catalog titles (can start with ~20 common ones + generic fallback).

- [ ] **Step 4: Pass + commit**

```bash
git add backend/internal/sense/science.go backend/internal/sense/*test.go
git commit -m "$(cat <<'EOF'
feat(sense): add PromptScience for 科普 concepts

EOF
)"
```

---

### Task 3: DB table `science_assets`

**Files:**
- Modify: `backend/internal/db/db.go` (after english DDL block)

- [ ] **Step 1: Add postgres + sqlite `CREATE TABLE IF NOT EXISTS science_assets`**

Copy `english_assets` DDL; rename table; replace `word_text` with `title VARCHAR(64)` (sqlite `TEXT`). Keep override column name `needs_sense_image_override` identical to english.

- [ ] **Step 2: Smoke migrate**

```bash
cd backend && go test ./internal/db/ -count=1
```

If no db tests, `go build ./...` is enough.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/db/db.go
git commit -m "$(cat <<'EOF'
feat(db): add science_assets table

EOF
)"
```

---

### Task 4: MinIO key helpers

**Files:**
- Modify: `backend/internal/storage/minio.go`

- [ ] **Step 1: Add**

```go
func ScienceGlyphObjectKey(kpID int64) string { return fmt.Sprintf("science/glyphs/%d.png", kpID) }
func ScienceSenseObjectKey(kpID int64) string { return fmt.Sprintf("science/senses/%d.png", kpID) }
func ScienceSpeechObjectKey(kpID int64) string { return fmt.Sprintf("science/speech/%d.mp3", kpID) }
```

Plus `(*ObjectStore) ScienceGlyphKey/SenseKey/SpeechKey` like english.

- [ ] **Step 2: `go build ./internal/storage/` + commit**

```bash
git add backend/internal/storage/minio.go
git commit -m "$(cat <<'EOF'
feat(storage): science MinIO object keys

EOF
)"
```

---

### Task 5: Science service core (Sync / List / Patch)

**Files:**
- Create: `backend/internal/science/service.go` (start with types + Sync/List/Patch; leave generate stubs if needed)
- Optional test with sqlite in-memory if english has a pattern; otherwise manual sync smoke later.

- [ ] **Step 1: Copy `english/service.go` structure**

Rename package `science`. Changes:
- `TableName` → `science_assets`
- Field `Title` / column `title` instead of `WordText`
- Sync SQL `WHERE s.code = ?` with `"science"`
- `NeedsSenseImage(row.Title)` always true
- DTOs: `ItemDTO` with `title` json; list path uses `items`
- Interfaces: same deps as english (`GlyphRenderer` with `RenderEnglishPNG`, images, TTS, store)

- [ ] **Step 2: Compile**

```bash
cd backend && go build ./internal/science/
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/science/
git commit -m "$(cat <<'EOF'
feat(science): sync/list/patch service mirroring english

EOF
)"
```

---

### Task 6: Generate glyph / sense / speech + batches

**Files:**
- Modify: `backend/internal/science/service.go`

- [ ] **Step 1: Port GenerateGlyph / BatchGenerateGlyphs**

Use `s.renderer.RenderEnglishPNG(asset.Title)` (supports 1–16 Han runes on white card). Store via `ScienceGlyphKey`. Public URL: `{APP_PUBLIC_BASE}/api/v1/science/items/{id}/glyph.png`.

- [ ] **Step 2: Port GenerateSense / BatchGenerateSenses**

`prompt := sense.PromptScience(asset.Title)`; skip when `!EffectiveNeedsSenseImage()`; worker pool identical to english (`workers` 1–8, retries ≤3). Prefer existing `listImageProviders` (Grok first) — either duplicate helper into science package or move shared helper to `sense`/`imagegen` if already shared; **do not** change english behavior. If `listImageProviders` is private in english, copy the function into science (YAGNI over premature extract) or unexport-sharing via `imagegen.ListProviders` only if a 5-line move is clean.

- [ ] **Step 3: Port speech generate/batch**

TTS text = `asset.Title`.

- [ ] **Step 4: `go test ./internal/science/ ./internal/sense/ -count=1` + `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add backend/internal/science/
git commit -m "$(cat <<'EOF'
feat(science): glyph/sense/speech generation and batches

EOF
)"
```

---

### Task 7: HTTP handlers + router + main

**Files:**
- Create: `backend/internal/http/handler_science.go` (copy `handler_english.go`, paths `/science/items/...`)
- Modify: `backend/internal/http/router.go` — `Deps.Science *science.Service` + routes
- Modify: `backend/cmd/server/main.go` — `science.NewService(...)`

Routes (match spec):

```
POST   /science/sync
GET    /science/items
PATCH  /science/items/:kpId
POST   /science/items/:kpId/glyph
GET    /science/items/:kpId/glyph.png
POST   /science/glyphs/batch
POST   /science/items/:kpId/sense
GET    /science/items/:kpId/sense.png
POST   /science/senses/batch
POST   /science/items/:kpId/speech
GET    /science/items/:kpId/speech.mp3
POST   /science/speech/batch
```

- [ ] **Step 1: Implement handlers + wire**

- [ ] **Step 2: Build**

```bash
cd backend && go build -o /tmp/sca-server ./cmd/server
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/http/handler_science.go backend/internal/http/router.go backend/cmd/server/main.go
git commit -m "$(cat <<'EOF'
feat(science): HTTP API and server wiring

EOF
)"
```

---

### Task 8: Frontend types + API client

**Files:**
- Create: `frontend/src/api/scienceTypes.ts` (from `englishTypes.ts`, `wordText` → `title`, names `ScienceItem`)
- Create: `frontend/src/api/science.ts` (paths `/api/v1/science/...`)

- [ ] **Step 1: Add files**

Include `batchGenerateSenses(moduleCode?, {workers, maxRetries})` even if UI unused (parity + ops).

- [ ] **Step 2: `npm run build` in frontend (or `tsc -b`) — expect fail until page exists if imports missing; only add client first.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/science.ts frontend/src/api/scienceTypes.ts
git commit -m "$(cat <<'EOF'
feat(science): frontend API client

EOF
)"
```

---

### Task 9: SciencePage + nav

**Files:**
- Create: `frontend/src/features/science/SciencePage.tsx` (clone `EnglishPage.tsx`)
- Modify: `frontend/src/App.tsx` — route `/science`
- Modify: `frontend/src/layout/AppShell.tsx` — NavLink **科普** after 英语

UI copy:
- eyebrow `CONTENT / SCIENCE`
- title `科普素材`
- description: 字图 / 义图 / 读音由后台生成…
- search placeholder: 搜索知识点
- card shows `item.title`

No batch buttons on page.

- [ ] **Step 1: Implement page + routes**

- [ ] **Step 2: Build**

```bash
cd frontend && npm run build
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/features/science frontend/src/App.tsx frontend/src/layout/AppShell.tsx
git commit -m "$(cat <<'EOF'
feat(science): 科普素材页 and top nav

EOF
)"
```

---

### Task 10: Docker rebuild + smoke

**Files:** none (ops)

- [ ] **Step 1: Rebuild stack**

```bash
cd /Users/conchi/workforce/go_workforce/study-content-admin && docker compose up --build -d
```

Ensure shared-config-center is up (Grok providers).

- [ ] **Step 2: Sync + list**

```bash
curl -sS -X POST http://localhost:19091/api/v1/science/sync | python3 -m json.tool
curl -sS 'http://localhost:19091/api/v1/science/items?view=groups' | python3 -c '
import sys,json
d=json.load(sys.stdin)
groups=d.get("groups") or []
print("groups", len(groups), "total", d.get("total"))
assert len(groups)>=1
# all effective needs sense
items=[i for g in groups for i in g.get("items") or g.get("words") or []]
# adapt to actual json field name from DTO
'
```

Align assertion with real JSON (`items` inside groups). Expect ~10 groups / ~99 items; `effectiveNeedsSenseImage` true.

- [ ] **Step 3: One glyph + one sense**

```bash
ID=$(curl -sS 'http://localhost:19091/api/v1/science/items?view=table' | python3 -c 'import sys,json;print(json.load(sys.stdin)["items"][0]["kpId"])')
curl -sS -m 120 -X POST "http://localhost:19091/api/v1/science/items/$ID/glyph" | python3 -m json.tool | head
curl -sS -m 180 -X POST "http://localhost:19091/api/v1/science/items/$ID/sense" | python3 -m json.tool | head
```

- [ ] **Step 4: UI smoke**

Open `http://localhost:19091/science` — nav 科普, sync works, cards render.

- [ ] **Step 5: Optional small commit only if smoke forced code fixes**

---

## Self-review

1. **Spec coverage:** nav, sync, list, override, glyph/sense/speech, batch API, always-need rule, no UI batch buttons — tasked.  
2. **No placeholders:** concrete paths, commands, code.  
3. **Consistency:** `title` field, `/science/items`, Grok via existing provider order, table `science_assets`.

---

## Execution handoff

Plan saved to `docs/superpowers/plans/2026-08-30-science-assets.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — this session with executing-plans checkpoints  

Which approach?
