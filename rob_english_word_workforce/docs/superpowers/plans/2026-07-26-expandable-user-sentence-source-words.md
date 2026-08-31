# Expandable User Sentence Source Words Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将“用户造句结果”改造成句子主行，展开后显示完整挖空内容和每个来源错词的答题明细。

**Architecture:** Go 接口继续分页查询 `sentence_cloze_item`，随后批量读取 `wrong_word_events`、`game_answer_detail/game_record` 与 `sentence_cloze_answer_record`，组装稳定的 `sourceWords` 嵌套结构，避免 N+1。React 将表格抽成独立组件管理多行展开状态，筛选、用户、页码或页大小变化时由 `resetKey` 清空展开状态。

**Tech Stack:** Go 1.25、Gin、GORM、PostgreSQL、React 19、TypeScript 5.9、Ant Design、Node 内置测试运行器。

---

## File Structure

- Create `word_select_dashboard/server/api/v1/system/sys_cloze_result_source_words.go`: 来源事件 DTO、批量查询和纯组装逻辑。
- Create `word_select_dashboard/server/api/v1/system/sys_cloze_result_source_words_test.go`: 覆盖游戏答错、挖空答错、丢失事件和历史记录降级。
- Modify `word_select_dashboard/server/api/v1/system/sys_cloze_result.go`: `ClozeResultItem` 增加 `sourceWords`，分页查询结束后一次性补全来源。
- Create `word_select_dashboard/web-react/src/features/clozeResultPresentation.ts`: 单词数量、入口/难度、耗时和追溯文案的纯展示函数。
- Create `word_select_dashboard/web-react/src/components/ClozeResultTable.tsx`: 句子主行、可访问展开按钮、上下文和来源单词明细。
- Create `word_select_dashboard/web-react/test/clozeResultPresentation.test.ts`: 展示函数的行为测试。
- Create `word_select_dashboard/web-react/test/clozeResultTable.contract.test.ts`: 主行/展开区字段与无障碍契约。
- Modify `word_select_dashboard/web-react/src/types/clozeResult.ts`: 新增 `ClozeSourceWord` 和 `sourceWords`。
- Modify `word_select_dashboard/web-react/src/App.tsx`: 使用新表格组件，移除旧八列和详情弹窗。
- Modify `word_select_dashboard/web-react/src/styles/app.css`: 主行、双行句子、徽标和九列展开区样式。

### Task 1: Backend source-word assembly

**Files:**
- Create: `word_select_dashboard/server/api/v1/system/sys_cloze_result_source_words_test.go`
- Create: `word_select_dashboard/server/api/v1/system/sys_cloze_result_source_words.go`
- Modify: `word_select_dashboard/server/api/v1/system/sys_cloze_result.go`

- [x] **Step 1: Write failing assembly tests**

Add table-driven tests that construct `ClozeResultItem` and lookup maps directly:

```go
func TestAssembleClozeSourceWordsGameEvent(t *testing.T) {
    item := ClozeResultItem{
        Words:                 []string{"momentum"},
        SourceEventIDs:        []int64{21},
        SourceAnswerDetailIDs: []int64{101},
        SourceRecordIDs:       []int64{11},
        SourceWordIDs:         []int64{501},
    }
    event := wrongWordEventRow{
        ID: 21, Source: "rob_english_word_back", SourceAnswerDetailID: 101,
        RecordID: 11, WordID: 501, Word: "momentum", WordDifficulty: 486,
        SelectedMeaning: "阻力", CorrectMeaning: "动力", CreatedAt: mustTime("2026-07-26T17:01:20Z"),
    }
    game := gameAnswerSourceRow{
        DetailID: 101, Mode: "solo_training", TrainingDifficultyGroup: "大学英语",
        TrainingDifficultyLevel: "cet4", AnswerTimeMs: 1200,
    }

    got := assembleClozeSourceWords(item, map[int64]wrongWordEventRow{21: event},
        map[int64]gameAnswerSourceRow{101: game}, nil)

    if got[0].Mode != "单人训练" || got[0].SelectedAnswer != "阻力" ||
        got[0].TraceStatus != "available" {
        t.Fatalf("unexpected source word: %#v", got[0])
    }
}
```

Also assert:

- `sentence_cloze_practice` resolves the synthetic answer record ID, uses `cost_ms`, and does not invent game difficulty.
- an indexed source event that cannot be loaded yields `traceStatus: "missing"` while preserving the indexed word and IDs.
- an item without source events yields `traceStatus: "historical"` and `traceText: "历史生成，无答题来源"` for every generated word.

- [x] **Step 2: Run tests and verify RED**

Run:

```bash
cd word_select_dashboard/server
go test ./api/v1/system -run 'TestAssembleClozeSourceWords' -count=1
```

Expected: FAIL because `wrongWordEventRow`, `assembleClozeSourceWords` and the new DTO do not exist.

- [x] **Step 3: Implement DTO and pure assembly**

Define:

```go
type ClozeSourceWord struct {
    Word                    string     `json:"word"`
    TraceStatus             string     `json:"traceStatus"`
    Source                  string     `json:"source"`
    SourceLabel             string     `json:"sourceLabel"`
    SourceEventID           int64      `json:"sourceEventId"`
    SourceAnswerDetailID    int64      `json:"sourceAnswerDetailId"`
    SourceRecordID          int64      `json:"sourceRecordId"`
    SourceWordID            int64      `json:"sourceWordId"`
    WrongTime               *time.Time `json:"wrongTime"`
    Mode                    string     `json:"mode"`
    DifficultyGroup         string     `json:"difficultyGroup"`
    DifficultyLevel         string     `json:"difficultyLevel"`
    WordDifficulty          *int       `json:"wordDifficulty"`
    AnswerTimeMs            *int64     `json:"answerTimeMs"`
    SelectedAnswer          string     `json:"selectedAnswer"`
    CorrectAnswer           string     `json:"correctAnswer"`
    TraceText               string     `json:"traceText"`
}
```

`assembleClozeSourceWords` must use array index only as a fallback for missing event rows. Loaded event rows are authoritative for source/detail/record/word IDs and answer text. Game mode mapping is exactly `match -> 正式匹配`, `solo_training -> 单人训练`; cloze source label is `句子挖空练习` and mode stays `-`.

- [x] **Step 4: Implement fixed-count batch enrichment**

Add `loadClozeSourceWords(selectDB, robDB *gorm.DB, items []ClozeResultItem) []ClozeResultItem`:

1. deduplicate all positive `SourceEventIDs`;
2. query `public.wrong_word_events WHERE id IN ?` once through `global.GVA_DB`;
3. query matching `game_answer_detail` plus `game_record` once through the rob database;
4. derive cloze answer record IDs using `source_answer_detail_id / 1000` and query `sentence_cloze_answer_record` once;
5. assemble every item; lookup errors degrade affected slots to `missing` and never fail the primary result page.

Update `Items`:

```go
items = loadClozeSourceWords(global.GVA_DB, db, items)
```

and add:

```go
SourceWords []ClozeSourceWord `json:"sourceWords"`
```

to `ClozeResultItem`.

- [x] **Step 5: Run backend tests and verify GREEN**

Run:

```bash
cd word_select_dashboard/server
gofmt -w api/v1/system/sys_cloze_result.go \
  api/v1/system/sys_cloze_result_source_words.go \
  api/v1/system/sys_cloze_result_source_words_test.go
go test ./api/v1/system -run 'TestAssembleClozeSourceWords' -count=1
go test ./api/v1/system -count=1
```

Expected: both test commands PASS.

### Task 2: Frontend presentation types and helpers

**Files:**
- Create: `word_select_dashboard/web-react/test/clozeResultPresentation.test.ts`
- Create: `word_select_dashboard/web-react/src/features/clozeResultPresentation.ts`
- Modify: `word_select_dashboard/web-react/src/types/clozeResult.ts`

- [x] **Step 1: Write failing presentation tests**

Test these exact behaviors:

```ts
test("sourceWordCount uses enriched rows and falls back to generated words", () => {
  assert.equal(sourceWordCount({ sourceWords: [{ word: "raw" }], words: ["raw", "stone"], word: "raw" }), 1);
  assert.equal(sourceWordCount({ sourceWords: [], words: ["raw", "stone"], word: "raw" }), 2);
});

test("entry/mode and library/difficulty stay in separate columns", () => {
  assert.equal(sourceWordEntryModeLabel({ sourceLabel: "游戏答题", mode: "单人训练" }),
    "游戏答题 / 单人训练");
  assert.equal(sourceWordDifficultyLabel({ difficultyGroup: "大学英语", difficultyLevel: "cet4" }),
    "大学英语 / cet4");
});

test("answer time and trace fallbacks remain explicit", () => {
  assert.equal(formatSourceAnswerTime(1200), "1.2s");
  assert.equal(formatSourceAnswerTime(undefined), "-");
  assert.equal(sourceWordTraceLabel({ traceStatus: "historical", traceText: "" }), "历史生成，无答题来源");
});
```

- [x] **Step 2: Run tests and verify RED**

Run:

```bash
cd word_select_dashboard/web-react
node --experimental-strip-types --test test/clozeResultPresentation.test.ts
```

Expected: FAIL because `clozeResultPresentation.ts` does not exist.

- [x] **Step 3: Add API types and minimal helpers**

Add the `ClozeSourceWord` TypeScript interface matching the Go JSON field names and add:

```ts
sourceWords: ClozeSourceWord[];
```

to `ClozeResultItem`. Implement only the tested pure helpers. Keep `sourceWordEntryModeLabel` and
`sourceWordDifficultyLabel` separate, render an `available` trace as
`事件 #eventId · 答题 #sourceAnswerDetailId · 记录 #sourceRecordId`, and render a `missing`
trace as `来源记录缺失`.

- [x] **Step 4: Run presentation tests and verify GREEN**

Run:

```bash
cd word_select_dashboard/web-react
node --experimental-strip-types --test test/clozeResultPresentation.test.ts
```

Expected: all presentation tests PASS.

### Task 3: Expandable sentence table

**Files:**
- Create: `word_select_dashboard/web-react/test/clozeResultTable.contract.test.ts`
- Create: `word_select_dashboard/web-react/src/components/ClozeResultTable.tsx`
- Modify: `word_select_dashboard/web-react/src/App.tsx`
- Modify: `word_select_dashboard/web-react/src/styles/app.css`

- [x] **Step 1: Write failing component contract tests**

Read the component source and assert it contains:

```ts
assert.match(source, /aria-expanded=\{expanded\}/);
assert.match(source, /来源单词/);
assert.match(source, /答错时间/);
assert.match(source, /入口 \/ 模式/);
assert.match(source, /来源追溯/);
assert.match(source, /item\.clozeSentence/);
assert.match(source, /item\.explanationZh/);
assert.match(source, /useEffect\(\(\) => \{[\s\S]*setExpandedIds\(new Set\(\)\)[\s\S]*\}, \[resetKey\]\)/);
assert.doesNotMatch(source, /\.join\(", "\)/);
```

Read `App.tsx` and assert it imports/renders `ClozeResultTable`, supplies a reset key containing user/filter/page/page-size, and no longer declares `detailItem`.

- [x] **Step 2: Run contract tests and verify RED**

Run:

```bash
cd word_select_dashboard/web-react
node --experimental-strip-types --test test/clozeResultTable.contract.test.ts
```

Expected: FAIL because the component does not exist.

- [x] **Step 3: Implement the component**

Render the eight main columns:

1. `#序号`
2. `时间`
3. `用户`
4. `句子 / 翻译`
5. `来源单词`
6. `来源`
7. `模型`
8. `展开`

The sentence cell renders English on at most two lines and Chinese on one line, each with a full tooltip. The source-word cell renders only a count badge such as `3 个词`.

Each main row is followed conditionally by one `colSpan={8}` row containing:

- full cloze sentence;
- full Chinese explanation;
- a horizontally scrollable nine-column source word table with the approved columns.

The component keeps `Set<number>` state, allows multiple expanded rows, uses a real button with `aria-expanded`, and clears the set in a `useEffect` when `resetKey` changes.

- [x] **Step 4: Replace old table and modal in App**

Build:

```ts
const resultResetKey = [
  selectedUser?.userId ?? "all",
  resultKeyword.trim(),
  resultPage,
  resultPageSize,
].join(":");
```

Pass `items`, `loading`, `page`, `pageSize`, and `resetKey` to `ClozeResultTable`. Remove the old eight-column mapping, the clickable comma-separated word list, `detailItem` state and the “生成结果详情” modal.

- [x] **Step 5: Add scoped responsive styles**

Replace the old nth-child widths with `.cloze-sentence-main-table` rules. Use:

- `.cloze-sentence-cell` for two-line English plus one-line translation;
- `.cloze-source-count` for the badge;
- `.cloze-source-detail` for the expanded panel;
- `.cloze-source-word-grid` with nine explicit `minmax` columns and horizontal overflow;
- distinct muted status styling for `historical` and warning styling for `missing`;
- a mobile media rule that retains horizontal scrolling rather than collapsing columns.

- [x] **Step 6: Run frontend tests and build**

Run:

```bash
cd word_select_dashboard/web-react
node --experimental-strip-types --test \
  test/clozeResultPresentation.test.ts \
  test/clozeResultTable.contract.test.ts
npm run build
```

Expected: tests PASS and TypeScript/Vite build exits 0.

### Task 4: Full verification and scope review

**Files:**
- Verify all files listed above.

- [x] **Step 1: Run complete backend verification**

```bash
cd word_select_dashboard/server
go test ./... -count=1
```

Expected: exit 0 with no failed Go packages.

- [ ] **Step 2: Run all frontend tests and production build**

```bash
cd word_select_dashboard/web-react
node --experimental-strip-types --test test/*.test.ts
npm run build
```

Expected: all Node tests PASS and production build exits 0.

Actual: production build and this feature's 6 tests pass. The complete Node suite is 71/72;
the remaining pre-existing CLI configuration contract still expects `deleteCLIProvider`, while
the existing implementation uses `deleteCurrentProvider`.

- [x] **Step 3: Inspect final diff**

```bash
git diff --check
git diff --stat
git status --short
```

Confirm that this feature touches only the plan, Go cloze-result API/tests, React cloze-result types/helpers/component/tests, `App.tsx`, and `app.css`; pre-existing CLI Runner changes remain unstaged and unmodified by this feature.
