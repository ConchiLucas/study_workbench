# Wrong Word Best Sentence Only Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the user wrong-word list use only `word_clean_best_sentence.sentence`, with no `word.sentence` fallback.

**Architecture:** Keep the existing `/api/wrong-words/events` response and Vue rendering contract. Simplify the Java SQL provider so its single lateral join resolves the best-sentence record by normalized word and returns either `best_sentence` or `none`; remove the dictionary-word lateral join entirely.

**Tech Stack:** Java 21, Spring Boot, MyBatis SQL provider, PostgreSQL, JUnit 5, Vue 3, Vitest, Vite

---

### Task 1: Enforce the Best-Sentence-Only SQL Contract

**Files:**
- Modify: `rob_english_word_back/src/test/java/com/robword/mapper/WrongWordQueueEventSqlProviderTest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java`

- [x] **Step 1: Write the failing SQL contract assertions**

Replace the existing source-priority assertions with:

```java
assertTrue(sql.contains("word_clean_best_sentence"));
assertTrue(sql.contains("best_example.sentence AS example_sentence"));
assertTrue(sql.contains("WHEN best_example.sentence IS NOT NULL THEN 'best_sentence'"));
assertTrue(sql.contains("ELSE 'none'"));
assertFalse(sql.contains("dictionary_example"));
assertFalse(sql.contains("FROM word exact_word"));
assertFalse(sql.contains("FROM word fallback_word"));
assertFalse(sql.contains("THEN 'word'"));
```

These assertions lock the approved rule: the event query never reads
`word.sentence`, even when a progress row still has `word_id`.

- [x] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn -Dtest=WrongWordQueueEventSqlProviderTest test
```

Expected: FAIL because the current SQL contains `dictionary_example`,
`FROM word exact_word`, and source value `word`.

- [x] **Step 3: Remove the original-word query and fallback**

Change the `active_words` projection to:

```sql
best_example.sentence AS example_sentence,
CASE
    WHEN best_example.sentence IS NOT NULL THEN 'best_sentence'
    ELSE 'none'
END AS example_source,
```

Delete the complete `dictionary_example` lateral join. Keep only:

```sql
LEFT JOIN LATERAL (
    SELECT NULLIF(BTRIM(best.sentence), '') AS sentence
    FROM word_clean_best_sentence best
    WHERE LOWER(BTRIM(best.word)) = progress.normalized_word
      AND NULLIF(BTRIM(best.sentence), '') IS NOT NULL
    ORDER BY best.id
    LIMIT 1
) best_example ON true
```

Do not change DTO fields, paging, sorting, keyword filtering, or the Vue response
shape.

- [x] **Step 4: Run focused backend tests and verify GREEN**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn -Dtest=WrongWordQueueEventSqlProviderTest,WrongWordControllerContractTest test
```

Expected: all focused tests PASS.

- [x] **Step 5: Commit the behavior change**

```bash
git add \
  rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java \
  rob_english_word_back/src/test/java/com/robword/mapper/WrongWordQueueEventSqlProviderTest.java
git commit -m "fix: use best sentences for wrong words"
```

### Task 2: Verify Database Semantics and Runtime

**Files:**
- Modify only if verification exposes a defect in Task 1 files.
- Do not modify deployment scripts or database data.

- [x] **Step 1: Execute the production SQL against PostgreSQL**

Extract `WrongWordQueueEventSqlProvider.selectEvents()` exactly as production
builds it and execute one page for user `conchi` in a read-only transaction.

Verify:

```text
Every non-null example_source is best_sentence.
No row reports example_source = word.
Missing best sentences return example_source = none and example_sentence = null.
The page has no duplicate progress_key.
The total remains the unfinished unique-word count.
Keyword filtering still checks the wrong word only.
```

- [x] **Step 2: Run the complete Java suite**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn test
```

Expected: all Java tests PASS. If sandbox networking blocks test-only random
localhost ports, rerun the identical command with host permission.

- [x] **Step 3: Run the complete user frontend suite and production build**

Run:

```bash
cd rob_english_word_front
npm test
npm run build
```

Expected: all Vitest tests PASS and `vue-tsc`/Vite build succeeds.

- [x] **Step 4: Restart applications without Docker**

Run:

```bash
cd /Users/conchi/workforce/rob_english_word_workforce
./restart_all_services.sh restart
```

Expected: application ports 6011 through 6018 report Ready. PostgreSQL, Redis,
and MinIO may remain existing Docker dependencies; no frontend or backend
application is started as a Docker container.

- [x] **Step 5: Inspect runtime status and fresh logs**

Verify:

```text
6011 /wrong-words returns HTTP 200.
6015, 6017, and 6018 health endpoints return HTTP 200.
No frontend/backend application container publishes ports 6011-6018.
No new Java ERROR or Word Agent traceback appears after restart.
```

The isolated browser may redirect to `/login`; do not guess credentials, reset
the password, or bypass authentication. Database execution, automated tests,
build output, and service health remain the required evidence when no signed-in
browser session is available.

- [x] **Step 6: Review the final diff**

Run:

```bash
git diff --check
git status --short
```

Confirm that pre-existing deployment-script changes and planning files remain
untouched by the feature commit.

- [x] **Step 7: Request code review**

Review the behavior commit against design commit `ba7563a`. Fix every Critical
or Important finding, rerun the focused RED/GREEN tests, then repeat the full
Java suite, frontend suite/build, SQL verification, runtime status, and
`git diff --check` before completion.
