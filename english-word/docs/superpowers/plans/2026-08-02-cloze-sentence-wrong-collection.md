# Cloze Sentence Wrong Collection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为“开始答题”和“单独训练”建立统一的句子错题集，使任一空位拼写错误都会激活整句进度，并按立即、7 天、15 天三阶段复习。

**Architecture:** 升级现有 `sentence_cloze_review_schedule` 为所有句源共用的持久句子进度，完成后保留 stage 3；`word-agent` 仍并行维护逐词进度。答题流水增加幂等键、实际入口和错误空位快照，到期接口合并句子进度与既有单词载体任务，用户端新增独立的可展开错题集覆盖层。

**Tech Stack:** PostgreSQL 15、Java 21、Spring Boot 3.2、MyBatis-Plus、JUnit 5/Mockito、React 19、TypeScript、Vitest、Testing Library。

---

## File map

**Create**

- `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentenceItem.java`：列表投影。
- `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentenceDetail.java`：展开详情投影。
- `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentenceAttempt.java`：最近作答摘要。
- `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentencePageResponse.java`：分页与汇总响应。
- `rob_english_word_back/src/main/java/com/robword/mapper/ClozeWrongSentenceQueryMapper.java`：列表、计数、详情查询。
- `rob_english_word_back/src/main/java/com/robword/service/ClozeAnswerComparison.java`：共享按空位比较结果。
- `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeReviewScheduleMapperContractTest.java`：整句原子状态机合同。
- `rob_english_word_back/src/test/java/com/robword/mapper/ClozeWrongSentenceQueryMapperContractTest.java`：列表与详情 SQL 合同。
- `rob_english_word_back/src/test/java/com/robword/controller/ClozeWrongSentenceControllerContractTest.java`：接口合同。
- `rob_english_word_cloze_web/src/components/WrongSentenceCollection.tsx`：错题集覆盖层。
- `rob_english_word_cloze_web/test/wrongSentenceCollection.test.tsx`：列表交互测试。

**Modify**

- `rob_english_word_back/db/sentence_cloze_review_schedule.sql`：持久完成状态、错误统计和幂等回填。
- `rob_english_word_back/db/sentence_cloze_answer_record.sql`：submission key、入口、操作类型和错误空位快照。
- `rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeReviewSchedule.java`：新增字段。
- `rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeAnswerRecord.java`：新增字段。
- `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeAnswerRequest.java`：新增提交元数据。
- `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeStatsResponse.java`：错句和精确到期数。
- `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeAnswerRecordMapper.java`：幂等查询和最近作答。
- `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeReviewScheduleMapper.java`：错误 upsert、到期正确原子推进。
- `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`：两类到期任务合并去重。
- `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`：共享比较、幂等返回、双进度更新、列表和详情。
- `rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java`：错题分页与详情接口。
- `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`：句子生命周期和幂等行为。
- `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java`：合并到期 SQL。
- `rob_english_word_cloze_web/src/types/cloze.ts`：错题 DTO 与新增统计字段。
- `rob_english_word_cloze_web/src/lib/api.ts`：列表、详情和提交字段。
- `rob_english_word_cloze_web/src/components/PracticeLaunchers.tsx`：错题集入口。
- `rob_english_word_cloze_web/src/App.tsx`：覆盖层状态、精确数量和 submission key。
- `rob_english_word_cloze_web/src/styles/app.css`：桌面展开表和移动卡片。
- `rob_english_word_cloze_web/test/practiceLaunchers.test.tsx`：首页三按钮合同。
- `rob_english_word_cloze_web/test/fullscreenCloseNavigation.test.ts`：错题集关闭层级。
- `rob_english_word_cloze_web/package.json`：纳入新测试。
- `docs/chains/cloze-practice-review.md`：更新权威链路与幂等说明。

## Task 1: Persist sentence progress and idempotent answer metadata

**Files:**

- Modify: `rob_english_word_back/db/sentence_cloze_review_schedule.sql`
- Modify: `rob_english_word_back/db/sentence_cloze_answer_record.sql`
- Modify: `rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeReviewSchedule.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeAnswerRecord.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeReviewScheduleMapper.java`
- Create: `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeReviewScheduleMapperContractTest.java`

- [x] **Step 1: Write the failing schema and mapper contract**

Assert the DDL and mapper SQL expose the durable lifecycle:

```java
assertTrue(schema.contains("status varchar(32) NOT NULL DEFAULT 'active'"));
assertTrue(schema.contains("wrong_count int4 NOT NULL DEFAULT 1"));
assertTrue(schema.contains("completed_time timestamp NULL"));
assertTrue(schema.contains("last_wrong_answer_record_id bigint NULL"));
assertTrue(schema.contains("submission_key varchar(64) NULL"));
assertTrue(schema.contains("wrong_blank_indexes_json text NOT NULL DEFAULT '[]'"));
assertTrue(wrongSql.contains("ON CONFLICT (user_id, cloze_item_id)"));
assertTrue(wrongSql.contains("wrong_count = sentence_cloze_review_schedule.wrong_count + 1"));
assertTrue(wrongSql.contains("status = 'active'"));
assertTrue(advanceSql.contains("next_review_time <= #{answeredAt}"));
assertTrue(advanceSql.contains("status = 'completed'"));
assertFalse(advanceSql.contains("DELETE FROM sentence_cloze_review_schedule"));
```

- [x] **Step 2: Run RED**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=SentenceClozeReviewScheduleMapperContractTest test
```

Expected: FAIL because the durable fields and atomic mapper methods do not exist.

- [x] **Step 3: Add the idempotent DDL**

Add columns with `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, relax `next_review_time` for completed rows, add checks and indexes:

```sql
status varchar(32) NOT NULL DEFAULT 'active',
wrong_count int4 NOT NULL DEFAULT 1,
first_wrong_time timestamp NULL,
last_wrong_answer_record_id bigint NULL,
completed_time timestamp NULL
```

```sql
ALTER TABLE public.sentence_cloze_review_schedule
    DROP CONSTRAINT IF EXISTS ck_sentence_cloze_review_status;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD CONSTRAINT ck_sentence_cloze_review_status
    CHECK (status IN ('active', 'completed'));
ALTER TABLE public.sentence_cloze_review_schedule
    ALTER COLUMN next_review_time DROP NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_review_user_status_time
    ON public.sentence_cloze_review_schedule(user_id, status, next_review_time);
```

Add nullable `submission_key`, `practice_context`, `action_type`, and non-null
`wrong_blank_indexes_json` to the answer table. Add a partial unique index:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS uk_sentence_cloze_answer_user_submission
    ON public.sentence_cloze_answer_record(user_id, submission_key)
    WHERE submission_key IS NOT NULL;
```

- [x] **Step 4: Add entity fields and atomic mapper methods**

Use these signatures:

```java
void upsertWrongSchedule(Long userId, Long clozeItemId, Long recordId,
                         LocalDateTime answeredAt);

int advanceDueCorrectSchedule(Long userId, Long clozeItemId, Long recordId,
                              LocalDateTime answeredAt,
                              LocalDateTime sevenDaysAt,
                              LocalDateTime fifteenDaysAt);
```

The correct update uses one `UPDATE ... CASE review_stage` statement. Stage 0 becomes 1,
stage 1 becomes 2, and stage 2 becomes 3/completed with a null due time. Its WHERE clause
requires `status='active'`, `next_review_time <= answeredAt`, and a different
`last_answer_record_id`.

- [x] **Step 5: Run GREEN and commit**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=SentenceClozeReviewScheduleMapperContractTest test
```

Expected: all contract assertions pass.

```bash
git add rob_english_word_back/db/sentence_cloze_review_schedule.sql \
  rob_english_word_back/db/sentence_cloze_answer_record.sql \
  rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeReviewSchedule.java \
  rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeAnswerRecord.java \
  rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeReviewScheduleMapper.java \
  rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeReviewScheduleMapperContractTest.java
git commit -m "feat: persist cloze sentence review progress"
```

## Task 2: Share answer comparison and make submission idempotent

**Files:**

- Create: `rob_english_word_back/src/main/java/com/robword/service/ClozeAnswerComparison.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeAnswerRequest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeAnswerRecordMapper.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/WrongWordReviewProgressService.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/service/WrongWordReviewProgressServiceTest.java`

- [x] **Step 1: Write RED tests for indexed comparison and idempotency**

Cover:

```java
var comparison = ClozeAnswerComparison.compare(
        List.of("raw", "", "fracture"),
        List.of("raw", "momentum", "fracture"));
assertFalse(comparison.correct());
assertEquals(List.of(1), comparison.wrongIndexes());
```

Submit a `word-agent` multi-blank sentence with one error and assert both calls occur:

```java
verify(reviewScheduleMapper).upsertWrongSchedule(7L, 93L, 302L, answeredAt);
verify(wrongWordReviewProgressService).applyAnswer(
        eq(7L), eq(item), eq(302L), eq(comparison), eq(answeredAt));
```

Mock `selectBySubmissionKey(7L, "submission-1")` with an existing record and assert no insert,
schedule update, word update, mastery update, or Agent notification occurs.

- [x] **Step 2: Run RED**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=ClozePracticeServiceTest,WrongWordReviewProgressServiceTest test
```

Expected: compilation or assertion failures for the missing comparison and submission metadata.

- [x] **Step 3: Implement the shared comparison**

`ClozeAnswerComparison` is an immutable record:

```java
public record ClozeAnswerComparison(
        List<String> answers,
        List<String> expectedWords,
        List<Integer> wrongIndexes,
        boolean correct
) {
    public static ClozeAnswerComparison compare(List<String> answers,
                                                List<String> expectedWords) {
        List<String> safeAnswers = answers == null ? List.of() : List.copyOf(answers);
        List<String> safeExpected = expectedWords == null ? List.of() : List.copyOf(expectedWords);
        List<Integer> wrongIndexes = new ArrayList<>();
        for (int index = 0; index < safeExpected.size(); index++) {
            String answer = index < safeAnswers.size() ? safeAnswers.get(index) : "";
            if (!normalize(answer).equals(normalize(safeExpected.get(index)))) {
                wrongIndexes.add(index);
            }
        }
        boolean correct = safeAnswers.size() == safeExpected.size() && wrongIndexes.isEmpty();
        return new ClozeAnswerComparison(
                safeAnswers, safeExpected, List.copyOf(wrongIndexes), correct);
    }

    private static String normalize(String value) {
        if (value == null) {
            return "";
        }
        return Normalizer.normalize(value.trim(), Normalizer.Form.NFKC)
                .toLowerCase(Locale.ROOT);
    }
}
```

The implementation pads missing positions with an empty string, preserves supplied array positions,
marks extra/missing positions as an incorrect sentence, and normalizes both sentence and word progress
through the same helper.

- [x] **Step 4: Implement submission reuse and dual progress updates**

Extend the request with `submissionKey`, `practiceContext`, and `actionType`. In
`submitAnswer`:

```java
SentenceClozeAnswerRecord replay = answerRecordMapper.selectBySubmissionKey(userId, submissionKey);
if (replay != null) {
    return toAnswerResponse(replay);
}
```

After a new answer row is inserted:

```java
if (!comparison.correct()) {
    reviewScheduleMapper.upsertWrongSchedule(userId, item.getId(), record.getId(), answeredAt);
} else {
    reviewScheduleMapper.advanceDueCorrectSchedule(
            userId, item.getId(), record.getId(), answeredAt,
            answeredAt.plusDays(7), answeredAt.plusDays(15));
}
if ("word-agent".equals(item.getSource())) {
    wrongWordReviewProgressService.applyAnswer(userId, item, record.getId(), comparison, answeredAt);
}
```

Persist `wrong_blank_indexes_json`, validated context (`review|solo`) and action
(`answer|reveal`). Duplicate-key insert races reload the existing submission and return it.

- [x] **Step 5: Run GREEN and commit**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=ClozePracticeServiceTest,WrongWordReviewProgressServiceTest test
```

Expected: all comparison, idempotency, sentence reset, word progress and mastery tests pass.

```bash
git add rob_english_word_back/src/main/java/com/robword/service/ClozeAnswerComparison.java \
  rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeAnswerRequest.java \
  rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeAnswerRecordMapper.java \
  rob_english_word_back/src/main/java/com/robword/service/WrongWordReviewProgressService.java \
  rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java \
  rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java \
  rob_english_word_back/src/test/java/com/robword/service/WrongWordReviewProgressServiceTest.java
git commit -m "feat: track wrong cloze sentences idempotently"
```

## Task 3: Merge due tasks and expose exact counts

**Files:**

- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeStatsResponse.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`

- [x] **Step 1: Write RED mapper and service tests**

Assert the due SQL contains both sources and deduplication:

```java
assertTrue(sql.contains("sentence_due AS"));
assertTrue(sql.contains("word_due AS"));
assertTrue(sql.contains("UNION ALL"));
assertTrue(sql.contains("GROUP BY cloze_item_id"));
assertTrue(sql.contains("MIN(next_review_time)"));
```

Assert stats return exact `activeWrongSentences` and `dueReviewTasks`, independent of the 10-item
batch limit.

- [x] **Step 2: Run RED**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=SentenceClozeItemMapperContractTest,ClozePracticeServiceTest test
```

Expected: FAIL because the existing query only reads word progress and stats lack the fields.

- [x] **Step 3: Implement one CTE-backed due projection**

Use `sentence_due`, `word_due`, `combined_due`, and `deduped_due` CTEs. Join the deduped IDs back
to `sentence_cloze_item`, order by `next_review_time, id`, and apply the requested limit only after
deduplication. Add a count query using the same CTE without `LIMIT`.

- [x] **Step 4: Run GREEN and commit**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=SentenceClozeItemMapperContractTest,ClozePracticeServiceTest test
```

Expected: merged task and exact-count tests pass.

```bash
git add rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java \
  rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeStatsResponse.java \
  rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java \
  rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java \
  rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java
git commit -m "feat: merge due cloze review tasks"
```

## Task 4: Add wrong sentence list and detail APIs

**Files:**

- Create: `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentenceItem.java`
- Create: `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentenceAttempt.java`
- Create: `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentenceDetail.java`
- Create: `rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentencePageResponse.java`
- Create: `rob_english_word_back/src/main/java/com/robword/mapper/ClozeWrongSentenceQueryMapper.java`
- Create: `rob_english_word_back/src/test/java/com/robword/mapper/ClozeWrongSentenceQueryMapperContractTest.java`
- Create: `rob_english_word_back/src/test/java/com/robword/controller/ClozeWrongSentenceControllerContractTest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java`

- [x] **Step 1: Write RED SQL and controller contracts**

The query contract must require progress as the driving table, status filtering, keyword binding,
stable sorting, last-wrong-record join, and user ownership. The controller contract requires:

```java
@GetMapping("/wrong-sentences")
ResponseEntity<ClozeWrongSentencePageResponse> getWrongSentences(...)

@GetMapping("/wrong-sentences/{progressId}")
ResponseEntity<ClozeWrongSentenceDetail> getWrongSentenceDetail(...)
```

- [x] **Step 2: Run RED**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=ClozeWrongSentenceQueryMapperContractTest,ClozeWrongSentenceControllerContractTest test
```

Expected: compilation fails because the DTO, mapper and endpoints do not exist.

- [x] **Step 3: Implement bounded paginated queries**

Normalize inputs in the service:

```java
String normalizedStatus = Set.of("active", "completed").contains(status) ? status : "active";
String normalizedSource = Set.of("all", "review", "solo").contains(source) ? source : "all";
String normalizedAvailability = Set.of("all", "due", "waiting").contains(availability)
        ? availability : "all";
int normalizedPage = Math.max(page == null ? 1 : page, 1);
int normalizedSize = Math.max(1, Math.min(size == null ? 20 : size, 100));
```

Map `practice_context` from the latest wrong record to the user-facing source. Decode
`wrong_blank_indexes_json` in Java. Detail lookup first verifies `(progress_id,user_id)`, then fetches
the latest five attempts ordered by `create_time DESC,id DESC`. Do not project original answer text,
model configuration or trace IDs.

- [x] **Step 4: Run GREEN and commit**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn -Dtest=ClozeWrongSentenceQueryMapperContractTest,ClozeWrongSentenceControllerContractTest test
```

Expected: mapper and controller contracts pass.

```bash
git add rob_english_word_back/src/main/java/com/robword/dto/ClozeWrongSentence*.java \
  rob_english_word_back/src/main/java/com/robword/mapper/ClozeWrongSentenceQueryMapper.java \
  rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java \
  rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java \
  rob_english_word_back/src/test/java/com/robword/mapper/ClozeWrongSentenceQueryMapperContractTest.java \
  rob_english_word_back/src/test/java/com/robword/controller/ClozeWrongSentenceControllerContractTest.java
git commit -m "feat: expose cloze wrong sentence collection"
```

## Task 5: Add the homepage entry and typed client API

**Files:**

- Modify: `rob_english_word_cloze_web/src/types/cloze.ts`
- Modify: `rob_english_word_cloze_web/src/lib/api.ts`
- Modify: `rob_english_word_cloze_web/src/components/PracticeLaunchers.tsx`
- Modify: `rob_english_word_cloze_web/test/practiceLaunchers.test.tsx`

- [x] **Step 1: Write the failing launcher test**

Render:

```tsx
<ReviewPracticeLauncher
  dueCount={3}
  wrongCount={18}
  loading={false}
  onStart={onStart}
  onOpenWrongSentences={onOpenWrongSentences}
  onOpenSolo={onOpenSolo}
/>
```

Assert buttons “开始答题”, “错题集 18”, and “单独训练” exist and invoke their callbacks.

- [x] **Step 2: Run RED**

```bash
cd rob_english_word_cloze_web
npx vitest run test/practiceLaunchers.test.tsx
```

Expected: TypeScript/render failure because the new props and button do not exist.

- [x] **Step 3: Add exact TypeScript contracts and API calls**

Define `WrongSentenceItem`, `WrongSentenceAttempt`, `WrongSentenceDetail`,
`WrongSentencePageResponse`, and query parameter types matching Task 4. Add:

```ts
export function getWrongSentences(token: string, query: WrongSentenceQuery) {
  return request<WrongSentencePageResponse>(
    `/api/cloze-practice/wrong-sentences?${new URLSearchParams(toQueryRecord(query))}`,
    {}, token,
  );
}

export function getWrongSentenceDetail(token: string, progressId: number) {
  return request<WrongSentenceDetail>(
    `/api/cloze-practice/wrong-sentences/${progressId}`, {}, token,
  );
}
```

Extend answer submission payload with `submissionKey`, `practiceContext`, and `actionType`.

- [x] **Step 4: Add the two-column secondary launcher row**

Keep the primary action full-width. Render the wrong collection and solo buttons inside one
`review-secondary-actions` wrapper. The wrong button remains enabled with count zero.

- [x] **Step 5: Run GREEN and commit**

```bash
cd rob_english_word_cloze_web
npx vitest run test/practiceLaunchers.test.tsx
```

Expected: all launcher tests pass.

```bash
git add rob_english_word_cloze_web/src/types/cloze.ts \
  rob_english_word_cloze_web/src/lib/api.ts \
  rob_english_word_cloze_web/src/components/PracticeLaunchers.tsx \
  rob_english_word_cloze_web/test/practiceLaunchers.test.tsx
git commit -m "feat: add cloze wrong collection entry"
```

## Task 6: Build the expandable wrong sentence collection

**Files:**

- Create: `rob_english_word_cloze_web/src/components/WrongSentenceCollection.tsx`
- Create: `rob_english_word_cloze_web/test/wrongSentenceCollection.test.tsx`
- Modify: `rob_english_word_cloze_web/src/App.tsx`
- Modify: `rob_english_word_cloze_web/src/styles/app.css`
- Modify: `rob_english_word_cloze_web/test/fullscreenCloseNavigation.test.ts`
- Modify: `rob_english_word_cloze_web/package.json`

- [x] **Step 1: Write RED component tests**

Mock list and detail loaders and verify:

```tsx
expect(screen.getByText("句子错题集")).toBeInTheDocument();
expect(screen.getByRole("tab", { name: "待复习" })).toHaveAttribute("aria-selected", "true");
expect(screen.getByText("立即复习")).toBeInTheDocument();
fireEvent.click(screen.getByRole("button", { name: "展开" }));
await screen.findByText("复习轨迹");
expect(screen.queryByText("你的答案")).not.toBeInTheDocument();
```

Add the close-navigation contract for `label="关闭错题集"` and require it to close before the
underlying review page changes.

- [x] **Step 2: Run RED**

```bash
cd rob_english_word_cloze_web
npx vitest run test/wrongSentenceCollection.test.tsx test/fullscreenCloseNavigation.test.ts
```

Expected: component import and close-contract failures.

- [x] **Step 3: Implement the focused component**

Props:

```ts
interface WrongSentenceCollectionProps {
  token: string;
  onClose: () => void;
  onAuthExpired: () => void;
}
```

The component owns status/source/availability/sort/page filters, abort-safe list loading,
per-row lazy detail loading, expanded IDs, retry actions and accessible tab/button labels. It renders
the eight desktop columns from the design and a semantic card per item under the mobile breakpoint.

- [x] **Step 4: Wire App state and idempotent submissions**

Add `showWrongSentences`. Open it from the launcher and render the component as a sibling fullscreen
overlay. Escape closes layers in this order: practice, answer results, sentence list, difficulty,
wrong collection, solo launcher.

Generate a submission key for each network attempt, retain it after an unknown network failure, and
clear it after a successful response. Send `practiceSource` as `practiceContext`; reveal sends
`actionType: "reveal"`, normal submit sends `"answer"`.

- [x] **Step 5: Add responsive styling**

Desktop uses an expandable table/card hybrid. Under 900px, hide the table header, render field labels
with `data-label`, stack sentence/translation and keep touch targets at least 44px high. Do not use a
fixed 980px minimum width.

- [x] **Step 6: Run GREEN and commit**

```bash
cd rob_english_word_cloze_web
npx vitest run test/wrongSentenceCollection.test.tsx test/practiceLaunchers.test.tsx test/fullscreenCloseNavigation.test.ts
npm run build
```

Expected: component/launcher/navigation tests and production build pass.

```bash
git add rob_english_word_cloze_web/src/components/WrongSentenceCollection.tsx \
  rob_english_word_cloze_web/src/App.tsx \
  rob_english_word_cloze_web/src/styles/app.css \
  rob_english_word_cloze_web/test/wrongSentenceCollection.test.tsx \
  rob_english_word_cloze_web/test/fullscreenCloseNavigation.test.ts \
  rob_english_word_cloze_web/package.json
git commit -m "feat: build cloze sentence wrong collection"
```

## Task 7: Migrate, document, verify, and deploy the workspace change

**Files:**

- Modify: `docs/chains/cloze-practice-review.md`
- Modify: `task_plan.md`
- Modify: `findings.md`
- Modify: `progress.md`

- [x] **Step 1: Add an executable migration behavior test**

Run both modified DDL scripts twice in a PostgreSQL transaction against temporary fixture tables.
Assert existing active schedules survive, latest-still-wrong rows backfill once, completed/ambiguous
history is not reactivated, and the partial submission-key unique index rejects a duplicate non-null key.

- [x] **Step 2: Run complete backend verification**

```bash
cd rob_english_word_back
JAVA_HOME=$(/usr/libexec/java_home -v 21) mvn test
```

Expected: all Java tests pass, including the PostgreSQL behavior test when the configured test database
is reachable.

- [x] **Step 3: Run complete frontend verification**

```bash
cd rob_english_word_cloze_web
npm test
npm run build
```

Expected: every Vitest file and the TypeScript/Vite production build pass.

- [x] **Step 4: Update the cross-project chain document**

Document the unified answer flow, sentence-vs-word progress boundary, merged due query, idempotent
submission key, wrong collection endpoints and exact stage rules. Remove the old statement that
submission idempotency is unverified.

- [ ] **Step 5: Apply the real database migration and workspace deployment**

Collect only the actually modified Workspace-relative paths and call
`apply_workspace_changes(task_id, changed_files)` when Context Router is available. Poll the returned
operation to a terminal state. If it remains unavailable, use the exact target project full/fast deploy
script selected by `deploy/context-router/README.md`; do not clean containers or data on failure.

- [ ] **Step 6: Runtime acceptance**

Verify:

- both incorrect entry modes create one active sentence row;
- the homepage exact due count matches the deduplicated query;
- wrong collection filters, expansion and mobile layout render;
- a repeated submission key returns the original result without changing counts;
- current service logs contain no new ERROR, panic or traceback after startup.

- [ ] **Step 7: Final review and commit**

Run `git diff --check`, review every changed path against the design, and commit only the remaining
documentation/test adjustments:

```bash
git add docs/chains/cloze-practice-review.md task_plan.md findings.md progress.md
git commit -m "docs: document cloze sentence review flow"
```
