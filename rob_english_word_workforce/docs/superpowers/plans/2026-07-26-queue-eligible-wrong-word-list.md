# Queue-Eligible Wrong Word List Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show every queue-eligible game or cloze wrong-word event in the user wrong-word list and remove the “你的答案” presentation column from all approved user/admin result tables.

**Architecture:** Add a backward-compatible Java endpoint backed by a PostgreSQL `UNION ALL` query that normalizes game and cloze wrong answers into one event DTO. Replace the Vue wrong-word aggregate cards with a paginated seven-column event view. Keep answer data in existing APIs and types, but stop rendering the selected-answer column in Vue and React tables.

**Tech Stack:** Java 21, Spring Boot 3.2, MyBatis Plus, PostgreSQL JSONB, Vue 3, TypeScript, Vitest, React 19, Node test runner, Vite

---

### Task 1: Lock the unified event SQL contract

**Files:**
- Create: `rob_english_word_back/src/main/java/com/robword/dto/WrongWordQueueEvent.java`
- Create: `rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/GameAnswerDetailMapper.java`
- Create: `rob_english_word_back/src/test/java/com/robword/mapper/WrongWordQueueEventSqlProviderTest.java`

- [ ] **Step 1: Write the failing SQL contract test**

Create a JUnit test that calls the provider methods and asserts the required source and eligibility clauses:

```java
@Test
void combinesEveryQueueEligibleGameAndClozeWrongWord() {
    String sql = WrongWordQueueEventSqlProvider.selectEvents();

    assertTrue(sql.contains("FROM game_answer_detail d"));
    assertTrue(sql.contains("d.is_correct = 0"));
    assertTrue(sql.contains("BTRIM(d.word_content) <> ''"));
    assertFalse(sql.contains("d.word_id IS NOT NULL"));
    assertTrue(sql.contains("FROM sentence_cloze_answer_record r"));
    assertTrue(sql.contains("jsonb_array_elements_text"));
    assertTrue(sql.contains("WITH ORDINALITY"));
    assertTrue(sql.contains("jsonb_array_length"));
    assertTrue(sql.contains("UNION ALL"));
    assertTrue(sql.contains("COUNT(*) OVER"));
    assertTrue(sql.contains("LIMIT #{size} OFFSET #{offset}"));
}

@Test
void countUsesTheSameNormalizedEventSet() {
    String sql = WrongWordQueueEventSqlProvider.countEvents();

    assertTrue(sql.contains("UNION ALL"));
    assertTrue(sql.contains("COUNT(*)"));
    assertTrue(sql.contains("#{keyword} IS NULL"));
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn -Dtest=WrongWordQueueEventSqlProviderTest test
```

Expected: test compilation fails because the provider does not exist.

- [ ] **Step 3: Add the normalized event DTO**

Create `WrongWordQueueEvent` with these fields and Lombok accessors:

```java
private String eventKey;
private String word;
private LocalDateTime answeredAt;
private String entry;
private String mode;
private String difficultyGroup;
private String difficultyLevel;
private String difficultyLabel;
private Integer wordDifficulty;
private Long costMs;
private String correctAnswer;
private String sourceType;
private Integer occurrenceCount;
```

- [ ] **Step 4: Implement one shared CTE provider**

Implement a provider with:

```java
public static String selectEvents()
public static String countEvents()
```

The shared CTE must:

- select all current-user `game_answer_detail` rows where `is_correct = 0` and trimmed `word_content` is non-empty, without a `word_id` requirement;
- derive the game correct answer from `correct_answer_index`;
- map match and solo-training record fields into entry, mode and difficulty metadata;
- expand `expected_words_json` with ordinality for each incorrect cloze target;
- treat unequal answer-array lengths as all-targets-wrong;
- compare equal-length answers case-insensitively after trimming;
- assign stable keys `game:<detailId>` and `cloze:<recordId>:<ordinal>`;
- calculate `occurrence_count` per normalized word;
- support keyword filtering, recent/count sorting, limit and offset.

Use fixed MyBatis placeholders rather than concatenating request values:

```sql
WHERE (#{keyword} IS NULL OR word ILIKE CONCAT('%', #{keyword}, '%'))
ORDER BY
  CASE WHEN #{sort} = 'count' THEN occurrence_count END DESC,
  answered_at DESC,
  event_key DESC
LIMIT #{size} OFFSET #{offset}
```

- [ ] **Step 5: Add mapper methods**

Add two `@SelectProvider` methods:

```java
List<WrongWordQueueEvent> selectQueueEligibleWrongWordEvents(
        Long userId, String keyword, String sort, Integer size, Long offset);

Long countQueueEligibleWrongWordEvents(Long userId, String keyword);
```

- [ ] **Step 6: Run the focused test and verify GREEN**

Run the same Maven command. Expected: both SQL contract tests pass.

### Task 2: Expose the backward-compatible event endpoint

**Files:**
- Modify: `rob_english_word_back/src/main/java/com/robword/controller/WrongWordController.java`
- Create: `rob_english_word_back/src/test/java/com/robword/controller/WrongWordControllerContractTest.java`

- [ ] **Step 1: Write the failing controller contract test**

Use reflection to assert that `listWrongWordEvents` exists, has `@GetMapping("/events")`, and accepts authentication plus keyword, sort, page and size parameters.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn -Dtest=WrongWordControllerContractTest test
```

Expected: FAIL because the endpoint method does not exist.

- [ ] **Step 3: Implement the endpoint**

Add:

```java
@GetMapping("/events")
public ResponseEntity<Map<String, Object>> listWrongWordEvents(
        Authentication auth,
        @RequestParam(required = false) String keyword,
        @RequestParam(defaultValue = "recent") String sort,
        @RequestParam(defaultValue = "1") Integer page,
        @RequestParam(defaultValue = "20") Integer size)
```

Reuse the existing keyword, sort, page and size normalization. Return `items`, `total`, `current` and `pages`. Keep the old aggregate and details endpoints unchanged.

- [ ] **Step 4: Run both backend focused tests**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn -Dtest=WrongWordQueueEventSqlProviderTest,WrongWordControllerContractTest test
```

Expected: all focused backend tests pass.

### Task 3: Replace the user wrong-word aggregate view

**Files:**
- Modify: `rob_english_word_front/src/views/WrongWordsView.vue`
- Create: `rob_english_word_front/src/views/WrongWordsView.contract.test.ts`

- [ ] **Step 1: Write the failing page contract test**

Read `WrongWordsView.vue` as raw source and assert:

```ts
expect(source).toContain("/api/wrong-words/events")
for (const label of ["来源单词", "答错时间", "入口 / 模式", "词库 / 难度", "词难度", "耗时", "正确答案"]) {
  expect(source).toContain(label)
}
expect(source).not.toContain("你的答案")
expect(source).not.toContain("/details")
expect(source).toContain("可入队错题记录")
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd rob_english_word_front
npm test -- WrongWordsView.contract.test.ts
```

Expected: FAIL because the view still uses the aggregate endpoint and cards.

- [ ] **Step 3: Implement the event table**

Replace the aggregate `WrongWord` and modal types with:

```ts
interface WrongWordEvent {
  eventKey: string
  word: string
  answeredAt: string
  entry: string
  mode: string
  difficultyGroup: string
  difficultyLevel: string
  difficultyLabel: string
  wordDifficulty: number | null
  costMs: number | null
  correctAnswer: string | null
  sourceType: 'game' | 'cloze'
  occurrenceCount: number
}
```

Fetch `/api/wrong-words/events`, preserve keyword, sort, page and size parameters, and render desktop headers plus narrow-screen labeled cards with exactly the seven approved fields. Remove the details request, modal state and modal markup.

Add presentation helpers:

```ts
formatEntryMode(item)
formatDifficulty(item)
formatCost(ms)
formatDateTime(value)
```

Use `-` for missing values and `暂无可入队错题记录` for the empty state.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run the same Vitest command. Expected: contract test passes.

### Task 4: Remove the user-answer column from the user frontend

**Files:**
- Modify: `rob_english_word_front/src/views/TrainingAnswerResultsView.vue`
- Create: `rob_english_word_front/src/views/answerColumnRemoval.contract.test.ts`

- [ ] **Step 1: Write the failing column-removal test**

Assert that `WrongWordsView.vue` and `TrainingAnswerResultsView.vue` do not contain `你的答案`, while both retain `正确答案`.

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
cd rob_english_word_front
npm test -- answerColumnRemoval.contract.test.ts
```

Expected: FAIL because training results still render the user-answer column.

- [ ] **Step 3: Remove the column**

Remove the training result header and row cell for `selectedAnswerText(detail)`, delete the unused helper, and change the desktop `.word-grid` from eight to seven columns. Keep `selectedAnswerIndex` in the response type.

- [ ] **Step 4: Run both user frontend focused tests**

Run:

```bash
cd rob_english_word_front
npm test -- WrongWordsView.contract.test.ts answerColumnRemoval.contract.test.ts
```

Expected: both tests pass.

### Task 5: Remove the user-answer column from every admin table

**Files:**
- Modify: `word_select_dashboard/web-react/src/components/ClozeResultTable.tsx`
- Modify: `word_select_dashboard/web-react/src/App.tsx`
- Modify: `word_select_dashboard/web-react/src/styles/app.css`
- Modify: `word_select_dashboard/web-react/test/clozeResultTable.contract.test.ts`
- Create: `word_select_dashboard/web-react/test/answerColumnRemoval.contract.test.ts`

- [ ] **Step 1: Change the existing sentence-table test to RED**

Replace the existing positive `你的答案` assertion with:

```ts
assert.doesNotMatch(source, /你的答案/);
assert.doesNotMatch(source, /word\\.selectedAnswer/);
assert.match(source, /正确答案/);
```

- [ ] **Step 2: Add a failing App-wide rendering contract**

Read `App.tsx` and assert it contains no `你的答案`, while still containing `正确答案`. Also assert the approved grid class names still exist so removing a cell cannot silently remove the tables.

- [ ] **Step 3: Run admin tests and verify RED**

Run:

```bash
cd word_select_dashboard/web-react
node --test test/clozeResultTable.contract.test.ts test/answerColumnRemoval.contract.test.ts
```

Expected: both tests fail on current user-answer headers/cells.

- [ ] **Step 4: Remove all approved React cells**

Remove the header and selected-answer row cell from:

- `ClozeResultTable`;
- standalone user wrong words;
- standalone user cloze wrong words;
- user training results;
- user-detail wrong-word modal;
- user-detail cloze-wrong modal.

Remove local `userAnswer` variables that become unused, but retain API/type fields.

- [ ] **Step 5: Adjust every grid**

Reduce:

- `.cloze-source-word-grid` from nine to eight columns and lower its minimum width;
- `.user-result-word-grid` from eight to seven columns;
- `.user-wrong-history-grid` from six to five columns;
- `.user-cloze-history-grid` from five to four columns;
- `.wrong-detail-head`/`.wrong-detail-row` from six to five columns;
- `.cloze-wrong-detail-grid` from five to four columns.

- [ ] **Step 6: Run admin focused tests and verify GREEN**

Run the same Node test command. Expected: both tests pass.

### Task 6: Full verification and runtime handoff

**Files:**
- Verify all files above
- Update: `task_plan.md`
- Update: `findings.md`
- Update: `progress.md`

- [ ] **Step 1: Run backend tests**

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home mvn test
```

Expected: all tests pass. If loopback test servers are blocked in the sandbox, rerun the identical command with approved host permissions.

- [ ] **Step 2: Run user frontend tests and build**

```bash
cd rob_english_word_front
npm test
npm run build
```

Expected: all tests and the production build pass.

- [ ] **Step 3: Run admin tests and build**

```bash
cd word_select_dashboard/web-react
node --test test/*.test.ts
npm run build
```

Expected: all tests and the production build pass.

- [ ] **Step 4: Validate the SQL against current read-only data**

Execute the provider-equivalent query for user `conchi` in a read-only PostgreSQL transaction. Confirm:

- the event count includes game wrong answers without requiring `word_id`;
- any current cloze wrong answer is expanded by wrong target word;
- the result columns and sort order are populated as designed;
- no source data is changed.

- [ ] **Step 5: Review the final diff**

Run:

```bash
git diff --check
git status --short
git diff --stat
```

Confirm the pre-existing immediate-review changes remain present and unchanged except for any unavoidable merge in the same Java test file.

- [ ] **Step 6: Restart without Docker**

Run the canonical restart with JDK 21 and explicit Docker-off flags:

```bash
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
START_DOCKER_DEPS=0 STOP_DOCKER_STACKS=0 \
./restart_all_services.sh restart
```

Verify services and ports 6011–6018 are ready.
