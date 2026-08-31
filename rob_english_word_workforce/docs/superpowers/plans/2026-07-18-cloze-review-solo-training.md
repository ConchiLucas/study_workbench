# Cloze Review and Solo Training Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the cloze launcher with a due-wrong-review home and a separate non-rank solo-training page whose last difficulty is persisted on the authenticated user.

**Architecture:** Java exposes explicit due-review and user preference APIs backed by two new `users` columns. React keeps launcher mode and difficulty in memory, loads both from the server after authentication, and renders separate launcher components while reusing the existing immersive answer overlay.

**Tech Stack:** Java 21, Spring Boot 3, MyBatis Plus, PostgreSQL, JUnit 5, Mockito, React 19, TypeScript 5, Vite 7, Vitest, Testing Library.

---

## File map

- Modify `rob_english_word_back/db/users.sql`: add and document solo difficulty columns with safe backfill.
- Modify `rob_english_word_back/src/main/java/com/robword/entity/User.java`: map the two preference fields.
- Create `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticePreferenceResponse.java`: preference response contract.
- Create `rob_english_word_back/src/main/java/com/robword/dto/UpdateSoloDifficultyRequest.java`: preference update request.
- Modify `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`: add strict due-wrong query.
- Modify `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`: due review and preference business rules.
- Modify `rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java`: expose GET due tasks and GET/PUT preference endpoints.
- Modify `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`: service red/green tests.
- Create `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java`: verify the annotated query keeps all required guards.
- Create `rob_english_word_cloze_web/src/lib/soloDifficulty.ts`: options, normalization, default and display helpers.
- Create `rob_english_word_cloze_web/src/lib/practiceMode.ts`: source-specific batch completion behavior.
- Create `rob_english_word_cloze_web/src/components/PracticeLaunchers.tsx`: review and solo launcher components.
- Modify `rob_english_word_cloze_web/src/lib/api.ts`: due-review and preference clients.
- Modify `rob_english_word_cloze_web/src/types/cloze.ts`: preference types.
- Modify `rob_english_word_cloze_web/src/App.tsx`: mode state, server initialization, source-specific batches and overlays.
- Modify `rob_english_word_cloze_web/src/styles/app.css`: A-layout styling and responsive rules.
- Modify `rob_english_word_cloze_web/package.json` and lockfile: add repeatable React tests.
- Create `rob_english_word_cloze_web/test/soloDifficulty.test.ts`: pure preference tests.
- Create `rob_english_word_cloze_web/test/practiceLaunchers.test.tsx`: launcher behavior tests.

### Task 1: User preference persistence

**Files:**
- Modify: `rob_english_word_back/db/users.sql`
- Modify: `rob_english_word_back/src/main/java/com/robword/entity/User.java`
- Create: `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticePreferenceResponse.java`
- Create: `rob_english_word_back/src/main/java/com/robword/dto/UpdateSoloDifficultyRequest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java`
- Test: `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`

- [ ] **Step 1: Write failing preference tests**

Add tests that stub `userMapper.selectById(7L)` and assert the default, stored and updated paths:

```java
@Test
void returnsDefaultSoloDifficultyWhenUserPreferenceIsInvalid() {
    User user = new User();
    user.setId(7L);
    user.setSoloDifficultyGroup("rank");
    user.setSoloDifficultyLevel("rank_current");
    when(userMapper.selectById(7L)).thenReturn(user);

    ClozePracticePreferenceResponse response = service.getPreferences(7L);

    assertEquals("junior", response.getSoloDifficultyGroup());
    assertEquals("junior", response.getSoloDifficultyLevel());
    verify(userMapper).updateById(argThat(updated ->
            "junior".equals(updated.getSoloDifficultyGroup())
                    && "junior".equals(updated.getSoloDifficultyLevel())));
}

@Test
void savesValidSoloDifficultyAndRejectsRankDifficulty() {
    User user = new User();
    user.setId(7L);
    when(userMapper.selectById(7L)).thenReturn(user);

    UpdateSoloDifficultyRequest valid = new UpdateSoloDifficultyRequest();
    valid.setDifficultyGroup("junior");
    valid.setDifficultyLevel("junior_7_1");
    assertEquals("junior_7_1", service.updateSoloDifficulty(7L, valid).getSoloDifficultyLevel());

    UpdateSoloDifficultyRequest rank = new UpdateSoloDifficultyRequest();
    rank.setDifficultyGroup("rank");
    rank.setDifficultyLevel("rank_current");
    assertThrows(IllegalArgumentException.class, () -> service.updateSoloDifficulty(7L, rank));
}
```

- [ ] **Step 2: Run the preference tests and confirm red**

Run: `cd rob_english_word_back && mvn -Dtest=ClozePracticeServiceTest test`

Expected: test compilation fails because preference DTOs and service methods do not exist.

- [ ] **Step 3: Add schema and entity fields**

Add both columns to the table definition and idempotent migration section:

```sql
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS solo_difficulty_group varchar(64) NOT NULL DEFAULT 'junior';
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS solo_difficulty_level varchar(64) NOT NULL DEFAULT 'junior';
UPDATE public.users
SET solo_difficulty_group = 'junior', solo_difficulty_level = 'junior'
WHERE solo_difficulty_group IS NULL OR solo_difficulty_group = ''
   OR solo_difficulty_level IS NULL OR solo_difficulty_level = '';
```

Add matching `@TableField` fields to `User` and column comments to `users.sql`.

- [ ] **Step 4: Implement DTOs and preference service rules**

Implement `getPreferences(Long userId)` and `updateSoloDifficulty(Long userId, UpdateSoloDifficultyRequest request)`. Validate a parent selection when `level.equals(group)`, otherwise require `DIFFICULTY_SOURCE_MAP.get(level)` to belong to `DIFFICULTY_GROUP_SOURCE_MAP.get(group)`. Reject all rank and unknown values. Persist only `id`, `soloDifficultyGroup` and `soloDifficultyLevel` through `userMapper.updateById`.

- [ ] **Step 5: Add authenticated controller endpoints**

Add:

```java
@GetMapping("/preferences")
public ResponseEntity<ClozePracticePreferenceResponse> getPreferences(Authentication auth)

@PutMapping("/preferences/solo-difficulty")
public ResponseEntity<ClozePracticePreferenceResponse> updateSoloDifficulty(
        Authentication auth,
        @RequestBody UpdateSoloDifficultyRequest request)
```

Both use `resolveUserId(auth)`.

- [ ] **Step 6: Run target tests**

Run: `cd rob_english_word_back && mvn -Dtest=ClozePracticeServiceTest test`

Expected: preference tests and existing best-sentence cloze test pass.

- [ ] **Step 7: Commit preference persistence**

Run:

```bash
git add rob_english_word_back/db/users.sql \
  rob_english_word_back/src/main/java/com/robword/entity/User.java \
  rob_english_word_back/src/main/java/com/robword/dto/ClozePracticePreferenceResponse.java \
  rob_english_word_back/src/main/java/com/robword/dto/UpdateSoloDifficultyRequest.java \
  rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java \
  rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java \
  rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java
git commit -m "feat: persist solo cloze difficulty"
```

### Task 2: Strict due-wrong review API

**Files:**
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`
- Create: `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java`

- [ ] **Step 1: Write failing service and mapper contract tests**

Service test:

```java
@Test
void returnsOnlyMapperDueReviewBatch() {
    SentenceClozeItem item = new SentenceClozeItem();
    item.setId(31L);
    when(sentenceClozeItemMapper.selectDueReviewItems(7L, 10)).thenReturn(List.of(item));

    List<ClozePracticeTaskResponse> result = service.getDueReviewTasks(7L, 10);

    assertEquals(List.of(31L), result.stream().map(ClozePracticeTaskResponse::getId).toList());
    verify(sentenceClozeItemMapper).selectDueReviewItems(7L, 10);
}
```

Mapper contract test reflects `@Select` on `selectDueReviewItems` and asserts the SQL contains all of:

```java
assertTrue(sql.contains("review.next_review_time <= NOW()"));
assertTrue(sql.contains("latest.is_correct = false"));
assertTrue(sql.contains("i.user_id = #{userId}"));
assertTrue(sql.contains("ORDER BY review.next_review_time ASC, i.id ASC"));
```

- [ ] **Step 2: Run tests and confirm red**

Run: `cd rob_english_word_back && mvn -Dtest=ClozePracticeServiceTest,SentenceClozeItemMapperContractTest test`

Expected: compilation fails because `selectDueReviewItems` and `getDueReviewTasks` are missing.

- [ ] **Step 3: Implement mapper query**

Add `selectDueReviewItems(userId, limit)` with an inner join to the user's review schedule and a lateral inner join to the latest answer. Require `review.next_review_time <= NOW()` and `latest.is_correct = false`; order by review time then item ID and apply the normalized limit.

- [ ] **Step 4: Implement service and controller**

Add `getDueReviewTasks` with the existing 1–100 limit normalization and `toTaskResponse`. Expose:

```java
@GetMapping("/tasks/review-due")
public ResponseEntity<List<ClozePracticeTaskResponse>> getDueReviewTasks(
        Authentication auth,
        @RequestParam(required = false) Integer limit)
```

- [ ] **Step 5: Run backend tests**

Run: `cd rob_english_word_back && mvn -Dtest=ClozePracticeServiceTest,SentenceClozeItemMapperContractTest test`

Expected: all focused tests pass.

- [ ] **Step 6: Commit due review API**

Run:

```bash
git add rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java \
  rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java \
  rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java \
  rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java \
  rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java
git commit -m "feat: expose due cloze reviews"
```

### Task 3: Frontend domain helpers, API and launchers

**Files:**
- Modify: `rob_english_word_cloze_web/package.json`
- Modify: `rob_english_word_cloze_web/package-lock.json`
- Create: `rob_english_word_cloze_web/src/lib/soloDifficulty.ts`
- Create: `rob_english_word_cloze_web/src/components/PracticeLaunchers.tsx`
- Modify: `rob_english_word_cloze_web/src/lib/api.ts`
- Modify: `rob_english_word_cloze_web/src/types/cloze.ts`
- Create: `rob_english_word_cloze_web/test/soloDifficulty.test.ts`
- Create: `rob_english_word_cloze_web/test/practiceLaunchers.test.tsx`

- [ ] **Step 1: Add test tooling and write failing frontend tests**

Add scripts and dev dependencies for Vitest, jsdom and Testing Library. Test the default and rank rejection:

```ts
expect(normalizeSoloDifficulty({ soloDifficultyGroup: "rank", soloDifficultyLevel: "rank_current" }))
  .toEqual(DEFAULT_SOLO_DIFFICULTY);
expect(SOLO_DIFFICULTY_OPTIONS.some((option) => option.key === "rank")).toBe(false);
```

Render `ReviewPracticeLauncher` and assert only “开始答题” and “单独训练” are present. Render `SoloTrainingLauncher` and assert “选择难度”“句子列表”“答题结果”“开始训练”“返回” are present.

- [ ] **Step 2: Run tests and confirm red**

Run: `cd rob_english_word_cloze_web && npm test`

Expected: tests fail because the helper and components do not exist.

- [ ] **Step 3: Implement pure difficulty module**

Move the existing non-rank parent/child options and types into `soloDifficulty.ts`. Export `DEFAULT_SOLO_DIFFICULTY` as `junior/junior`, `normalizeSoloDifficulty`, `selectedDifficultyText` and option lookup helpers. Do not access `localStorage` in this module.

- [ ] **Step 4: Implement API contracts**

Add:

```ts
export interface ClozePracticePreference {
  soloDifficultyGroup: string;
  soloDifficultyLevel: string;
}

export function getDueReviewTasks(token: string, limit = 10): Promise<ClozePracticeTask[]>
export function getClozePreferences(token: string): Promise<ClozePracticePreference>
export function updateSoloDifficulty(token: string, difficultyGroup: string, difficultyLevel: string): Promise<ClozePracticePreference>
```

- [ ] **Step 5: Implement launcher components**

`ReviewPracticeLauncher` receives `dueCount`, `loading`, `onStart`, and `onOpenSolo`. Disable start when loading or `dueCount === 0`. `SoloTrainingLauncher` receives selected label, batch text and five callbacks. Both retain existing CSS button classes where useful.

- [ ] **Step 6: Run frontend focused tests**

Run: `cd rob_english_word_cloze_web && npm test`

Expected: all helper and launcher tests pass.

- [ ] **Step 7: Commit frontend domain and launchers**

Run:

```bash
git add rob_english_word_cloze_web/package.json rob_english_word_cloze_web/package-lock.json \
  rob_english_word_cloze_web/src/lib/soloDifficulty.ts \
  rob_english_word_cloze_web/src/components/PracticeLaunchers.tsx \
  rob_english_word_cloze_web/src/lib/api.ts rob_english_word_cloze_web/src/types/cloze.ts \
  rob_english_word_cloze_web/test/soloDifficulty.test.ts \
  rob_english_word_cloze_web/test/practiceLaunchers.test.tsx
git commit -m "feat: add cloze review and solo launchers"
```

### Task 4: App mode orchestration and visual styling

**Files:**
- Modify: `rob_english_word_cloze_web/src/App.tsx`
- Modify: `rob_english_word_cloze_web/src/styles/app.css`
- Create: `rob_english_word_cloze_web/src/lib/practiceMode.ts`
- Modify: `rob_english_word_cloze_web/test/practiceLaunchers.test.tsx`
- Create: `rob_english_word_cloze_web/test/practiceMode.test.ts`

- [ ] **Step 1: Add failing mode behavior tests**

Extend launcher tests to click buttons and assert callbacks. Add `nextLauncherModeAfterBatch(source)` in `practiceMode.ts`, returning `"review"` for a review batch and `"solo"` for a solo batch, and test both values without mounting the entire 1,800-line app.

- [ ] **Step 2: Run tests and confirm red**

Run: `cd rob_english_word_cloze_web && npm test`

Expected: new callback or mode helper assertions fail before orchestration is implemented.

- [ ] **Step 3: Replace browser difficulty persistence**

Delete `DIFFICULTY_STORAGE_KEY`, `readStoredDifficulty`, and `persistDifficulty`. Initialize state with `DEFAULT_SOLO_DIFFICULTY`. After authentication, load preferences, due tasks and stats in parallel; normalize the server preference into memory only.

- [ ] **Step 4: Add launcher and practice source state**

Add `launcherMode: "review" | "solo"` and `practiceSource: "review" | "solo"`. Review start activates the already loaded due batch. Solo start calls the existing difficulty batch endpoint. When a batch ends, review mode refreshes due tasks and returns home; solo mode continues its selected difficulty behavior. Escape/close returns to the source launcher.

- [ ] **Step 5: Persist difficulty before applying selection**

Make parent and child selection handlers async. Call `updateSoloDifficulty`; only after success set `selectedDifficulty`, close the picker and clear the old batch. On failure retain the old selection and show the API message.

- [ ] **Step 6: Replace launcher JSX and filter picker**

Render `ReviewPracticeLauncher` on the home mode and `SoloTrainingLauncher` on solo mode. Remove the rank card and all rank selection handlers. Keep sentence-list and answer-result overlays reachable only from the solo launcher.

- [ ] **Step 7: Add A-layout CSS**

Style a focused two-button review launcher, solo header with return action, disabled/empty review state and responsive stacking. Reuse the existing dark palette and purple primary gradient; preserve sidebar and immersive overlay behavior.

- [ ] **Step 8: Run frontend tests and build**

Run: `cd rob_english_word_cloze_web && npm test && npm run build`

Expected: all tests pass and TypeScript/Vite build succeeds.

- [ ] **Step 9: Commit app orchestration and layout**

Run:

```bash
git add rob_english_word_cloze_web/src/App.tsx rob_english_word_cloze_web/src/styles/app.css \
  rob_english_word_cloze_web/src/lib/practiceMode.ts \
  rob_english_word_cloze_web/test/practiceLaunchers.test.tsx \
  rob_english_word_cloze_web/test/practiceMode.test.ts
git commit -m "feat: split cloze review and solo training flows"
```

### Task 5: Database migration, full verification and runtime QA

**Files:**
- Modify: `progress.md`
- Modify: `findings.md`

- [ ] **Step 1: Apply the idempotent users migration**

Run the updated `rob_english_word_back/db/users.sql` against local `rob_english_word`. Verify both columns are non-null and existing users contain `junior/junior` unless they already have a valid saved preference.

- [ ] **Step 2: Run all automated checks**

Run:

```bash
cd rob_english_word_back && mvn test
cd rob_english_word_cloze_web && npm test && npm run build
git diff --check
```

Expected: Java, frontend tests, build and whitespace checks pass.

- [ ] **Step 3: Restart affected services**

Restart Java 8019/9091 and cloze web 7003 using their existing scripts. Confirm ports are ready without invoking paid LLM/TTS generation.

- [ ] **Step 4: Browser QA**

At `http://localhost:7003`, verify logged-in desktop and narrow viewport flows: review-only home, due empty/ready states, solo navigation, no rank option, server-restored preference, sentence list/results access, both practice sources and correct return destination.

- [ ] **Step 5: Record evidence**

Append exact migration counts, test totals, build result, service status and browser observations to `progress.md` and `findings.md`.

- [ ] **Step 6: Commit verification records**

Run:

```bash
git add progress.md findings.md
git commit -m "docs: record cloze launcher verification"
```
