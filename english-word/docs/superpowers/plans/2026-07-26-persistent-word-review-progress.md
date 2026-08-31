# Persistent Word Review Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将错题集改为按用户、单词持久维护的未完成复习集合，并让每个错词独立完成立即、7 天、15 天三轮后才移除。

**Architecture:** 在 `rob_english_word` 新增 `wrong_word_review_progress` 事实表和 Java 领域服务。普通与挖空错误负责激活/重置，Word Agent 造句保存时关联当前题目，挖空提交按下标逐词推进；用户端和 Go 管理后台的主错题集以未完成进度为过滤源，事件历史只提供最近一次展示字段。

**Tech Stack:** PostgreSQL 15、Spring Boot 3、MyBatis-Plus、JUnit 5/Mockito、Vue 3/TypeScript/Vitest、Go/GORM、React/TypeScript。

---

## File map

**Create**

- `rob_english_word_back/db/wrong_word_review_progress.sql`：幂等建表、索引、触发器和历史回填。
- `rob_english_word_back/src/main/java/com/robword/entity/WrongWordReviewProgress.java`：单词级复习实体。
- `rob_english_word_back/src/main/java/com/robword/mapper/WrongWordReviewProgressMapper.java`：原子激活、关联、推进和列表查询。
- `rob_english_word_back/src/main/java/com/robword/service/WrongWordReviewProgressService.java`：标准化、逐词状态机和句子关联。
- `rob_english_word_back/src/test/java/com/robword/mapper/WrongWordReviewProgressMapperContractTest.java`：SQL 状态机合同。
- `rob_english_word_back/src/test/java/com/robword/service/WrongWordReviewProgressServiceTest.java`：逐词生命周期单元测试。

**Modify**

- `rob_english_word_back/src/main/java/com/robword/service/GameSettlementService.java`：普通答错激活进度。
- `rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java`：造句保存后关联三个词。
- `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`：逐空推进和完成判定。
- `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`：Word Agent 任务按单词进度到期。
- `rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java`：事件流水改为未完成唯一单词。
- `rob_english_word_back/src/main/java/com/robword/dto/WrongWordQueueEvent.java`：稳定进度键和复习状态字段。
- `rob_english_word_back/src/main/java/com/robword/controller/WrongWordController.java`：接口文案和 DTO 语义改为未完成词。
- 现有 Java 三个服务测试和 Mapper/Controller 合同测试：扩展 RED/GREEN 覆盖。
- `rob_english_word_front/src/views/WrongWordsView.vue`：未完成错词文案和稳定 key。
- `rob_english_word_front/src/views/WrongWordsView.contract.test.ts`：唯一单词口径合同。
- `word_select_dashboard/server/api/v1/system/sys_app_user.go`：管理后台主错题集按进度表过滤。
- `word_select_dashboard/web-react/src/types/user.ts`、`word_select_dashboard/web-react/src/App.tsx`：接收进度字段并保持现有七字段/无“你的答案”布局。

## Task 1: Add the persistent word-level schema and atomic state API

**Files:**

- Create: `rob_english_word_back/db/wrong_word_review_progress.sql`
- Create: `rob_english_word_back/src/main/java/com/robword/entity/WrongWordReviewProgress.java`
- Create: `rob_english_word_back/src/main/java/com/robword/mapper/WrongWordReviewProgressMapper.java`
- Create: `rob_english_word_back/src/test/java/com/robword/mapper/WrongWordReviewProgressMapperContractTest.java`

- [ ] **Step 1: Write the failing mapper contract test**

Create a test that reflects mapper annotations and asserts the SQL contains:

```java
assertTrue(upsertSql.contains("ON CONFLICT (user_id, normalized_word)"));
assertTrue(upsertSql.contains("wrong_count = wrong_word_review_progress.wrong_count + 1"));
assertTrue(upsertSql.contains("review_stage = 0"));
assertTrue(upsertSql.contains("completed_time = NULL"));
assertTrue(linkSql.contains("active_cloze_item_id = #{clozeItemId}"));
assertTrue(advanceSql.contains("next_review_time <= #{answeredAt}"));
assertTrue(advanceSql.contains("review_stage = 3"));
assertTrue(advanceSql.contains("status = 'completed'"));
```

- [ ] **Step 2: Run RED**

Run:

```bash
cd rob_english_word_back
mvn -Dtest=WrongWordReviewProgressMapperContractTest test
```

Expected: compilation fails because the mapper does not exist.

- [ ] **Step 3: Add the idempotent SQL schema**

Define:

```sql
CREATE TABLE IF NOT EXISTS public.wrong_word_review_progress (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    word_id bigint NULL,
    word varchar(100) NOT NULL,
    normalized_word varchar(100) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'waiting_sentence',
    review_stage int4 NOT NULL DEFAULT 0,
    next_review_time timestamp NULL,
    active_cloze_item_id bigint NULL,
    active_blank_index int4 NULL,
    wrong_count int4 NOT NULL DEFAULT 1,
    first_wrong_time timestamp NOT NULL,
    last_wrong_time timestamp NOT NULL,
    last_answer_record_id bigint NULL,
    completed_time timestamp NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_wrong_word_review_status
        CHECK (status IN ('waiting_sentence', 'due', 'waiting', 'completed')),
    CONSTRAINT ck_wrong_word_review_stage CHECK (review_stage BETWEEN 0 AND 3)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_wrong_word_review_user_word
    ON public.wrong_word_review_progress(user_id, normalized_word);
CREATE INDEX IF NOT EXISTS idx_wrong_word_review_user_status_time
    ON public.wrong_word_review_progress(user_id, status, next_review_time);
CREATE INDEX IF NOT EXISTS idx_wrong_word_review_active_item
    ON public.wrong_word_review_progress(active_cloze_item_id, active_blank_index);
```

Add the standard update-time trigger and an idempotent history backfill using
`ON CONFLICT (user_id, normalized_word) DO NOTHING`. Backfill game errors and
per-index cloze errors; attach the newest `word-agent` sentence containing the same
normalized blank word and set it `due`, otherwise `waiting_sentence`.

- [ ] **Step 4: Add entity and atomic mapper methods**

The mapper API must be:

```java
void upsertWrong(
        Long userId, Long wordId, String word, String normalizedWord,
        LocalDateTime wrongTime, Long answerRecordId);

void linkActiveSentence(
        Long userId, String normalizedWord, Long clozeItemId,
        Integer blankIndex, LocalDateTime dueTime);

int advanceDueCorrect(
        Long userId, Long clozeItemId, Integer blankIndex,
        Long answerRecordId, LocalDateTime answeredAt,
        LocalDateTime sevenDaysAt, LocalDateTime fifteenDaysAt);

List<WrongWordReviewProgress> selectByActiveItem(Long userId, Long clozeItemId);
```

`advanceDueCorrect` uses one PostgreSQL `UPDATE ... CASE` and only advances rows
whose `next_review_time <= answeredAt`. Stage 2 becomes stage 3/completed; stage 0
and 1 become waiting with the supplied 7/15-day timestamp.

- [ ] **Step 5: Run GREEN**

Run:

```bash
cd rob_english_word_back
mvn -Dtest=WrongWordReviewProgressMapperContractTest test
```

Expected: all mapper contract assertions pass.

- [ ] **Step 6: Commit Task 1 only**

```bash
git add rob_english_word_back/db/wrong_word_review_progress.sql \
  rob_english_word_back/src/main/java/com/robword/entity/WrongWordReviewProgress.java \
  rob_english_word_back/src/main/java/com/robword/mapper/WrongWordReviewProgressMapper.java \
  rob_english_word_back/src/test/java/com/robword/mapper/WrongWordReviewProgressMapperContractTest.java
git commit -m "feat: add persistent wrong word review progress"
```

## Task 2: Implement the word-level lifecycle service

**Files:**

- Create: `rob_english_word_back/src/main/java/com/robword/service/WrongWordReviewProgressService.java`
- Create: `rob_english_word_back/src/test/java/com/robword/service/WrongWordReviewProgressServiceTest.java`

- [ ] **Step 1: Write RED tests for normalization, linking, and independent progress**

Cover these behaviors with Mockito:

```java
service.recordWrong(7L, 11L, "  Momentum ", wrongAt, 301L);
verify(mapper).upsertWrong(7L, 11L, "Momentum", "momentum", wrongAt, 301L);

service.linkGeneratedSentence(7L, 91L, List.of("raw", "momentum", "fracture"), dueAt);
verify(mapper).linkActiveSentence(7L, "raw", 91L, 0, dueAt);
verify(mapper).linkActiveSentence(7L, "momentum", 91L, 1, dueAt);
verify(mapper).linkActiveSentence(7L, "fracture", 91L, 2, dueAt);
```

For `applyAnswer`, create three progress rows at indexes 0/1/2 and submit
`["raw", "wrong", "fracture"]`. Assert correct indexes call
`advanceDueCorrect`, while only index 1 calls `upsertWrong`.

- [ ] **Step 2: Run RED**

```bash
cd rob_english_word_back
mvn -Dtest=WrongWordReviewProgressServiceTest test
```

Expected: compilation fails because the service does not exist.

- [ ] **Step 3: Implement the minimal service**

Use:

```java
private String normalizeWord(String value) {
    return value == null ? "" : value.trim().toLowerCase(Locale.ROOT);
}
```

Public methods:

```java
void recordWrong(Long userId, Long wordId, String word,
                 LocalDateTime wrongAt, Long answerRecordId);
void linkGeneratedSentence(Long userId, Long clozeItemId,
                           List<String> words, LocalDateTime dueAt);
WordReviewUpdateResult applyAnswer(Long userId, SentenceClozeItem item,
                                   Long answerRecordId, List<String> expected,
                                   List<String> answers, LocalDateTime answeredAt);
```

`WordReviewUpdateResult` contains the words completed by this submission so mastery
updates occur only after stage 2 succeeds. Array-size mismatch marks every expected
index wrong, matching the queue notification rule.

- [ ] **Step 4: Run GREEN**

```bash
cd rob_english_word_back
mvn -Dtest=WrongWordReviewProgressServiceTest test
```

Expected: all lifecycle tests pass.

- [ ] **Step 5: Commit Task 2 only**

```bash
git add rob_english_word_back/src/main/java/com/robword/service/WrongWordReviewProgressService.java \
  rob_english_word_back/src/test/java/com/robword/service/WrongWordReviewProgressServiceTest.java
git commit -m "feat: implement per-word review lifecycle"
```

## Task 3: Activate progress from every wrong-answer entry and link generated sentences

**Files:**

- Modify: `rob_english_word_back/src/main/java/com/robword/service/GameSettlementService.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/service/GameSettlementServiceTest.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/service/SentenceClozeServiceTest.java`

- [ ] **Step 1: Write RED tests**

In `GameSettlementServiceTest`, inject a mocked progress service and save one wrong
detail plus one correct detail. Verify only the wrong detail calls:

```java
verify(progressService).recordWrong(
    eq(userId), eq(wordId), eq("momentum"), any(LocalDateTime.class), eq(detailId));
```

In `SentenceClozeServiceTest`, generate the three-word sentence and verify:

```java
verify(progressService).linkGeneratedSentence(
    eq(1L), eq(savedItemId),
    eq(List.of("brisk", "anchor", "harbor")),
    any(LocalDateTime.class));
```

Also verify repeated generation-key lookup links the existing item idempotently.

- [ ] **Step 2: Run RED**

```bash
cd rob_english_word_back
mvn -Dtest=GameSettlementServiceTest,SentenceClozeServiceTest test
```

Expected: verification failures because neither service calls progress yet.

- [ ] **Step 3: Implement activation and linking**

In `GameSettlementService`, call `recordWrong` after a wrong detail is inserted and
before the async Word Agent notification. Keep the optional injected dependency
pattern used by the notification service so existing constructors remain stable.

In `SentenceClozeService`, inject the progress service and call
`linkGeneratedSentence` after insert or generation-key conflict resolution. Linking
an existing generation key must be safe and must not increment `wrong_count`.

- [ ] **Step 4: Run GREEN**

```bash
cd rob_english_word_back
mvn -Dtest=GameSettlementServiceTest,SentenceClozeServiceTest test
```

Expected: both test classes pass.

- [ ] **Step 5: Commit Task 3 only**

```bash
git add rob_english_word_back/src/main/java/com/robword/service/GameSettlementService.java \
  rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java \
  rob_english_word_back/src/test/java/com/robword/service/GameSettlementServiceTest.java \
  rob_english_word_back/src/test/java/com/robword/service/SentenceClozeServiceTest.java
git commit -m "feat: activate review progress from wrong answers"
```

## Task 4: Advance each blank independently and fix 7/15-day task selection

**Files:**

- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java`

- [ ] **Step 1: Write RED tests**

Add service tests for:

- first due correct schedules 7 days and does not complete;
- second due correct schedules 15 days;
- third due correct completes;
- `["correct", "wrong", "correct"]` only resets the middle word;
- a correct non-due word does not advance;
- a wrong non-due word resets immediately and is notified;
- mastery is recorded only for words returned as newly completed.

Add mapper contract assertions:

```java
assertTrue(sql.contains("wrong_word_review_progress"));
assertTrue(sql.contains("p.next_review_time <= NOW()"));
assertFalse(sql.contains("latest.is_correct = false"));
```

- [ ] **Step 2: Run RED**

```bash
cd rob_english_word_back
mvn -Dtest=ClozePracticeServiceTest,SentenceClozeItemMapperContractTest test
```

Expected: new per-word verifications fail and mapper still references the latest
whole-sentence answer.

- [ ] **Step 3: Implement per-word submission**

For `word-agent` items:

```java
WordReviewUpdateResult update = progressService.applyAnswer(
    userId, item, record.getId(), expectedWords, answers, LocalDateTime.now());
```

Notify Word Agent for actual wrong indexes as before. Do not call the old whole-item
schedule for these items. Mark mastery only for `update.completedWords()`.

Keep the existing whole-item behavior for non-Word-Agent independent content until
it produces a queued Word Agent sentence.

- [ ] **Step 4: Update task SQL**

For Word Agent items, select a sentence when:

```sql
EXISTS (
    SELECT 1
    FROM wrong_word_review_progress p
    WHERE p.user_id = #{userId}
      AND p.active_cloze_item_id = i.id
      AND p.status <> 'completed'
      AND p.next_review_time <= NOW()
)
```

Do not require the latest answer to be wrong. Preserve the existing best-sentence
branches.

- [ ] **Step 5: Run GREEN**

```bash
cd rob_english_word_back
mvn -Dtest=ClozePracticeServiceTest,SentenceClozeItemMapperContractTest test
```

Expected: all stage, partial-answer, and due-query tests pass.

- [ ] **Step 6: Commit Task 4 only**

```bash
git add rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java \
  rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java \
  rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java \
  rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java
git commit -m "feat: advance cloze review per word"
```

## Task 5: Change the user wrong-word API from event history to active unique words

**Files:**

- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/WrongWordQueueEvent.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/controller/WrongWordController.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/mapper/WrongWordQueueEventSqlProviderTest.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/controller/WrongWordControllerContractTest.java`

- [ ] **Step 1: Write RED contracts**

Require the provider to:

```java
assertTrue(sql.contains("JOIN wrong_word_review_progress progress"));
assertTrue(sql.contains("progress.status <> 'completed'"));
assertTrue(sql.contains("ROW_NUMBER() OVER"));
assertTrue(sql.contains("PARTITION BY queue_events.user_id, LOWER(BTRIM(queue_events.word))"));
assertTrue(countSql.contains("COUNT(*)"));
assertFalse(countSql.contains("COUNT(*) FROM ranked_events"));
```

Controller behavior must return one DTO per progress row and total active unique
words, while retaining `/api/wrong-words/events` for frontend compatibility.

- [ ] **Step 2: Run RED**

```bash
cd rob_english_word_back
mvn -Dtest=WrongWordQueueEventSqlProviderTest,WrongWordControllerContractTest test
```

Expected: SQL assertions fail because the provider still returns every event.

- [ ] **Step 3: Implement active-word projection**

Keep the existing game/cloze event CTE to obtain the seven display fields, then:

```sql
active_progress AS (
    SELECT *
    FROM wrong_word_review_progress
    WHERE user_id = #{userId}
      AND status <> 'completed'
),
latest_event AS (
    SELECT queue_events.*,
           ROW_NUMBER() OVER (
             PARTITION BY queue_events.user_id, LOWER(BTRIM(queue_events.word))
             ORDER BY answered_at DESC, event_key DESC
           ) AS row_no
    FROM queue_events
)
```

Join `active_progress.normalized_word` to `latest_event` row 1. Map
`occurrence_count` from `progress.wrong_count`; use `progress.id` for the stable
key and return optional review fields.

- [ ] **Step 4: Run GREEN**

```bash
cd rob_english_word_back
mvn -Dtest=WrongWordQueueEventSqlProviderTest,WrongWordControllerContractTest test
```

Expected: mapper and endpoint contracts pass.

- [ ] **Step 5: Commit Task 5 only**

```bash
git add rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java \
  rob_english_word_back/src/main/java/com/robword/dto/WrongWordQueueEvent.java \
  rob_english_word_back/src/main/java/com/robword/controller/WrongWordController.java \
  rob_english_word_back/src/test/java/com/robword/mapper/WrongWordQueueEventSqlProviderTest.java \
  rob_english_word_back/src/test/java/com/robword/controller/WrongWordControllerContractTest.java
git commit -m "feat: list unfinished wrong words"
```

## Task 6: Update user and admin presentation contracts

**Files:**

- Modify: `rob_english_word_front/src/views/WrongWordsView.vue`
- Modify: `rob_english_word_front/src/views/WrongWordsView.contract.test.ts`
- Modify: `word_select_dashboard/server/api/v1/system/sys_app_user.go`
- Modify: `word_select_dashboard/web-react/src/types/user.ts`
- Modify: `word_select_dashboard/web-react/src/App.tsx`
- Modify: `word_select_dashboard/web-react/test/answerColumnRemoval.contract.test.ts`

- [ ] **Step 1: Write RED frontend contracts**

Assert the user page says:

```ts
expect(source).toContain('待复习单词')
expect(source).toContain('未完成复习的错词会持续显示')
expect(source).toContain(':key="item.progressKey"')
expect(source).not.toContain('可入队错题记录')
```

Keep the seven-column and no-“你的答案” assertions.

Add a Go/React source contract requiring the admin `WrongWords` SQL to join
`wrong_word_review_progress` and filter `status <> 'completed'`.

- [ ] **Step 2: Run RED**

```bash
cd rob_english_word_front
npm test -- --run src/views/WrongWordsView.contract.test.ts src/views/answerColumnRemoval.contract.test.ts

cd ../word_select_dashboard/web-react
npm test -- --runInBand
```

Expected: new active-progress strings are absent.

- [ ] **Step 3: Implement the presentation**

User page keeps the existing seven columns, search, sort, page size and responsive
cards. Rename `WrongWordEvent` to `PendingWrongWord`, use `progressKey`, and change
empty/header copy to persistent-review wording.

Admin `WrongWords` uses `wrong_word_review_progress` as the base set, joins the
latest ordinary or cloze event for metadata, and keeps historical expansion.
Expose `reviewStatus`, `reviewStage`, `nextReviewTime` in types without adding a
new visible column. Keep all previous “你的答案” removals.

- [ ] **Step 4: Run GREEN and builds**

```bash
cd rob_english_word_front
npm test -- --run
npm run build

cd ../word_select_dashboard/web-react
npm test
npm run build
```

Expected: user tests/build pass; admin task contracts and build pass. If the known
unrelated `deleteCLIProvider` baseline remains the sole full-suite failure, record
it without changing model configuration code.

- [ ] **Step 5: Commit Task 6 only**

```bash
git add rob_english_word_front/src/views/WrongWordsView.vue \
  rob_english_word_front/src/views/WrongWordsView.contract.test.ts \
  word_select_dashboard/server/api/v1/system/sys_app_user.go \
  word_select_dashboard/web-react/src/types/user.ts \
  word_select_dashboard/web-react/src/App.tsx \
  word_select_dashboard/web-react/test/answerColumnRemoval.contract.test.ts
git commit -m "feat: show persistent pending wrong words"
```

## Task 7: Apply migration, run full verification, and restart without Docker

**Files:**

- Verify: all files above
- Update: `task_plan.md`, `findings.md`, `progress.md`

- [ ] **Step 1: Validate and apply the SQL idempotently**

Use the existing Word Agent virtual environment database driver:

```bash
word_select_dashboard/word-agent/.venv/bin/python -c "
from pathlib import Path
import psycopg
sql = Path('rob_english_word_back/db/wrong_word_review_progress.sql').read_text()
with psycopg.connect('host=127.0.0.1 port=5432 dbname=rob_english_word user=conchi password=conchi123456') as conn:
    conn.execute(sql)
"
```

Run it twice. Expected: both runs succeed; the second run does not change row counts
or increment `wrong_count`.

- [ ] **Step 2: Verify data invariants**

Query:

```sql
SELECT user_id, normalized_word, COUNT(*)
FROM wrong_word_review_progress
GROUP BY user_id, normalized_word
HAVING COUNT(*) > 1;
```

Expected: zero rows.

Also verify every active row has stage 0–2, completed rows have stage 3, and each
active item/index points to the same normalized word in `blank_words_json`.

- [ ] **Step 3: Run full backend tests**

```bash
cd rob_english_word_back
mvn test
```

Expected: all Java tests pass. If the sandbox blocks the test HTTP server, rerun
this exact Maven test command with the required host permission and record why.

- [ ] **Step 4: Run Go tests and static checks**

```bash
cd word_select_dashboard/server
go test ./...

cd ../..
git diff --check
```

Expected: Go tests and whitespace checks pass.

- [ ] **Step 5: Restart every service without Docker**

Use the existing root non-Docker restart script. Do not start Docker or Compose.
Verify ports 6011–6018 and health endpoints, then scan only the current startup log
segments for `ERROR`, `Exception`, and `Traceback`.

- [ ] **Step 6: Runtime lifecycle smoke test**

With a test user and one synthetic word:

1. insert/submit one eligible wrong answer;
2. verify the word immediately appears once in `/api/wrong-words/events`;
3. wait for/link a generated sentence;
4. submit correct at stage 0 and verify the word remains with stage 1;
5. temporarily advance `next_review_time` for test data only, submit stage 1 correct,
   and verify stage 2;
6. temporarily advance again, submit stage 2 correct, and verify the word disappears;
7. submit the word wrong again and verify it reappears at stage 0.

Remove only synthetic test rows by exact IDs after recording results; do not alter
real user history.

- [ ] **Step 7: Final review**

Run the verification-before-completion checklist, inspect `git diff --stat`, confirm
no Docker processes were started, and update the three planning files with exact
test counts, runtime ports and any preserved baseline failures.

