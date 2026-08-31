# Wrong-Word Example Sentence Highlight Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one English example sentence to every user-facing unfinished wrong-word row, preferring the original word-library sentence, falling back to the AI best sentence, and safely highlighting the target word or phrase.

**Architecture:** Extend the existing `/api/wrong-words/events` SQL projection so pagination returns the selected sentence and a diagnostic source in the same query. Keep matching and highlighting in a small pure TypeScript helper, then let `WrongWordsView.vue` render its segments with Vue interpolation and semantic `<mark>` elements. No schema migration, per-row API call, or review-state change is required.

**Tech Stack:** Java 21, Spring Boot, MyBatis SQL Provider, PostgreSQL, Vue 3, TypeScript, Vitest, Vite

---

## Working-directory constraint

The user explicitly requested development in the current directory. Do not create a worktree. Preserve all pre-existing modified deployment scripts and planning files; every feature commit must stage only the files named by that task.

### Task 1: Project example sentences in the wrong-word API

**Files:**
- Modify: `rob_english_word_back/src/test/java/com/robword/mapper/WrongWordQueueEventSqlProviderTest.java`
- Modify: `rob_english_word_back/src/test/java/com/robword/controller/WrongWordControllerContractTest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/WrongWordQueueEvent.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java`

- [x] **Step 1: Write failing DTO and SQL contract tests**

Add a DTO reflection assertion:

```java
assertEquals(String.class, WrongWordQueueEvent.class
        .getDeclaredField("exampleSentence").getType());
assertEquals(String.class, WrongWordQueueEvent.class
        .getDeclaredField("exampleSource").getType());
```

Extend `projectsLatestMetadataForEveryUnfinishedUniqueWord()` with stable SQL-contract assertions:

```java
assertTrue(sql.contains("progress.word_id"));
assertTrue(sql.contains("word_clean_best_sentence"));
assertTrue(sql.contains("example_sentence"));
assertTrue(sql.contains("example_source"));
assertTrue(sql.contains("WHEN dictionary_example.sentence IS NOT NULL THEN 'word'"));
assertTrue(sql.contains("WHEN best_example.sentence IS NOT NULL THEN 'best_sentence'"));
assertTrue(sql.contains("ELSE 'none'"));
```

Also assert that the existing `LIMIT #{size} OFFSET #{offset}` and unfinished-word predicates remain present.

- [x] **Step 2: Run the focused Java tests and verify RED**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn -Dtest=WrongWordQueueEventSqlProviderTest,WrongWordControllerContractTest test
```

Expected: FAIL because `WrongWordQueueEvent` has no example fields and the SQL does not project example sentences.

- [x] **Step 3: Add the response fields**

Add to `WrongWordQueueEvent`:

```java
private String exampleSentence;
private String exampleSource;
```

- [x] **Step 4: Select the deterministic dictionary sentence**

In the `active_words` query, add one `LEFT JOIN LATERAL` named `dictionary_example`. Its candidate set must:

```sql
SELECT exact_word.sentence, 0 AS priority,
       exact_word.status, exact_word.difficulty, exact_word.frequency, exact_word.id
FROM word exact_word
WHERE exact_word.id = progress.word_id

UNION ALL

SELECT fallback_word.sentence, 1 AS priority,
       fallback_word.status, fallback_word.difficulty,
       fallback_word.frequency, fallback_word.id
FROM word fallback_word
WHERE NOT EXISTS (
    SELECT 1 FROM word original_word WHERE original_word.id = progress.word_id
)
  AND LOWER(BTRIM(fallback_word.word)) = progress.normalized_word
```

Order the candidates by:

```sql
priority,
CASE WHEN status = 1 THEN 0 ELSE 1 END,
COALESCE(difficulty, 2147483647),
COALESCE(frequency, 2147483647),
id
LIMIT 1
```

Normalize the selected sentence with `NULLIF(BTRIM(sentence), '')` so blank strings count as missing. The exact `word_id` row remains authoritative even if it was later disabled.

- [x] **Step 5: Add the AI-best fallback and final projection**

Add a second `LEFT JOIN LATERAL` named `best_example`:

```sql
SELECT NULLIF(BTRIM(best.sentence), '') AS sentence
FROM word_clean_best_sentence best
WHERE LOWER(BTRIM(best.word)) = progress.normalized_word
  AND NULLIF(BTRIM(best.sentence), '') IS NOT NULL
ORDER BY best.id
LIMIT 1
```

Project the final fields in `active_words` and the outer `SELECT`:

```sql
COALESCE(dictionary_example.sentence, best_example.sentence) AS example_sentence,
CASE
    WHEN dictionary_example.sentence IS NOT NULL THEN 'word'
    WHEN best_example.sentence IS NOT NULL THEN 'best_sentence'
    ELSE 'none'
END AS example_source
```

Do not add either field to the keyword predicate or count query.

- [x] **Step 6: Run focused Java tests and verify GREEN**

Run the same Maven command from Step 2.

Expected: all focused tests PASS.

- [x] **Step 7: Commit the backend contract**

```bash
git add \
  rob_english_word_back/src/test/java/com/robword/mapper/WrongWordQueueEventSqlProviderTest.java \
  rob_english_word_back/src/test/java/com/robword/controller/WrongWordControllerContractTest.java \
  rob_english_word_back/src/main/java/com/robword/dto/WrongWordQueueEvent.java \
  rob_english_word_back/src/main/java/com/robword/mapper/WrongWordQueueEventSqlProvider.java
git commit -m "feat: return examples for wrong words"
```

### Task 2: Implement safe sentence segmentation

**Files:**
- Create: `rob_english_word_front/src/lib/highlightSentence.ts`
- Create: `rob_english_word_front/src/lib/highlightSentence.test.ts`

- [x] **Step 1: Write the failing highlighter tests**

Create tests for exact matches, case-insensitive matches, phrases, all occurrences, substring rejection, regex characters, empty inputs, and missing matches:

```ts
import { describe, expect, it } from 'vitest'
import { splitHighlightedSentence } from './highlightSentence'

const highlightedText = (sentence: string, word: string) =>
  splitHighlightedSentence(sentence, word)
    .filter((segment) => segment.highlighted)
    .map((segment) => segment.text)

describe('splitHighlightedSentence', () => {
  it('highlights a target while preserving sentence case', () => {
    expect(highlightedText('This Word is useful.', 'word')).toEqual(['Word'])
  })

  it('highlights a multi-word phrase', () => {
    expect(highlightedText('Do not make a mistake today.', 'make a mistake'))
      .toEqual(['make a mistake'])
  })

  it('highlights every legal occurrence', () => {
    expect(highlightedText('Word by word.', 'word')).toEqual(['Word', 'word'])
  })

  it('does not match inside a longer alphanumeric word', () => {
    expect(highlightedText('Draw the raw material.', 'raw')).toEqual(['raw'])
  })

  it('escapes regular-expression characters', () => {
    expect(highlightedText('Use c++ carefully.', 'c++')).toEqual(['c++'])
  })

  it('returns one plain segment for empty or unmatched targets', () => {
    expect(splitHighlightedSentence('A sentence.', '')).toEqual([
      { text: 'A sentence.', highlighted: false },
    ])
    expect(splitHighlightedSentence('A sentence.', 'word')).toEqual([
      { text: 'A sentence.', highlighted: false },
    ])
  })
})
```

- [x] **Step 2: Run the helper test and verify RED**

Run:

```bash
cd rob_english_word_front
npm test -- src/lib/highlightSentence.test.ts
```

Expected: FAIL because `highlightSentence.ts` does not exist.

- [x] **Step 3: Implement the pure helper**

Export:

```ts
export interface HighlightSegment {
  text: string
  highlighted: boolean
}

export function splitHighlightedSentence(
  sentence?: string | null,
  word?: string | null,
): HighlightSegment[]
```

Implementation requirements:

1. Return `[]` when `sentence` is empty.
2. Trim the target word; return one plain segment when it is empty.
3. Escape `.*+?^${}()|[\]\\`.
4. Find case-insensitive occurrences while preserving original sentence text.
5. If the target begins or ends with an ASCII letter/digit, reject a match whose corresponding neighboring character is also an ASCII letter/digit.
6. Preserve rejected matches as ordinary text and continue searching.
7. Return one plain segment if no legal match was found.

- [x] **Step 4: Run the helper test and verify GREEN**

Run the same Vitest command from Step 2.

Expected: all helper cases PASS.

- [x] **Step 5: Commit the helper**

```bash
git add \
  rob_english_word_front/src/lib/highlightSentence.ts \
  rob_english_word_front/src/lib/highlightSentence.test.ts
git commit -m "feat: highlight words in example sentences"
```

### Task 3: Render the example column in the wrong-word page

**Files:**
- Modify: `rob_english_word_front/src/views/WrongWordsView.contract.test.ts`
- Modify: `rob_english_word_front/src/views/WrongWordsView.vue`
- Modify: `rob_english_word_front/src/lib/highlightSentence.ts`

- [x] **Step 1: Write the failing page contract**

Extend the approved-column list with `例句`, then add:

```ts
expect(source).toContain('exampleSentence')
expect(source).toContain('exampleSource')
expect(source).toContain('splitHighlightedSentence')
expect(source).toContain('class="example-highlight"')
expect(source).toContain('data-label="例句"')
expect(source).not.toContain('v-html')
```

- [x] **Step 2: Run the page contract and verify RED**

Run:

```bash
cd rob_english_word_front
npm test -- src/views/WrongWordsView.contract.test.ts
```

Expected: FAIL because the page has no example fields, column, or highlighter rendering.

- [x] **Step 3: Extend the frontend response type and template**

Add to `WrongWordEvent`:

```ts
exampleSentence: string | null
exampleSource: 'word' | 'best_sentence' | 'none'
```

Import `splitHighlightedSentence`. Add the “例句” header immediately after “来源单词”. Render:

```vue
<span
  class="example-sentence"
  data-label="例句"
  :title="item.exampleSentence || ''"
>
  <template v-if="item.exampleSentence">
    <template
      v-for="(segment, index) in splitHighlightedSentence(item.exampleSentence, item.word)"
      :key="`${item.progressKey}:example:${index}`"
    >
      <mark v-if="segment.highlighted" class="example-highlight">{{ segment.text }}</mark>
      <span v-else>{{ segment.text }}</span>
    </template>
  </template>
  <template v-else>—</template>
</span>
```

The template must not render `exampleSource`.

- [x] **Step 4: Update the desktop and responsive layout**

Increase `.page-header`, `.toolbar`, and `.wrong-container` maximum width from `1180px` to `1440px`.

Change `.event-grid` to eight columns:

```css
grid-template-columns:
  minmax(110px, 0.8fr) minmax(260px, 2fr) minmax(130px, 0.9fr)
  minmax(150px, 1.1fr) minmax(140px, 1fr) 72px 72px
  minmax(160px, 1.2fr);
```

Add:

```css
.example-sentence {
  display: -webkit-box;
  overflow: hidden;
  color: #d7dae3;
  line-height: 1.55;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.example-highlight {
  border-radius: 3px;
  padding: 0 2px;
  background: rgba(137, 92, 246, 0.18);
  color: #b99cff;
  font-weight: 700;
}
```

Use `@media (max-width: 1279px)` for the card-layout breakpoint so the eight-column table never overflows at intermediate widths. Preserve the existing one-column mobile breakpoint.

- [x] **Step 5: Run focused frontend tests and verify GREEN**

Run:

```bash
cd rob_english_word_front
npm test -- \
  src/lib/highlightSentence.test.ts \
  src/views/WrongWordsView.contract.test.ts \
  src/views/answerColumnRemoval.contract.test.ts \
  src/views/fullscreenNavigation.contract.test.ts
```

Expected: all focused tests PASS and no test expects the removed “你的答案” column.

- [x] **Step 6: Run the production frontend build**

Run:

```bash
cd rob_english_word_front
npm run build
```

Expected: `vue-tsc -b` and `vite build` complete successfully.

- [x] **Step 7: Commit the user-facing page**

```bash
git add \
  rob_english_word_front/src/lib/highlightSentence.ts \
  rob_english_word_front/src/views/WrongWordsView.contract.test.ts \
  rob_english_word_front/src/views/WrongWordsView.vue
git commit -m "feat: show examples in wrong-word list"
```

### Task 4: Verify SQL semantics, regressions, and non-Docker runtime

**Files:**
- Modify only if verification reveals a feature defect in files already listed above.
- Do not modify deployment scripts.

- [x] **Step 1: Execute the production SQL against PostgreSQL**

Use the existing local database configuration without printing credentials. Extract `WrongWordQueueEventSqlProvider.selectEvents()` exactly as production builds it, replace MyBatis parameters with typed PostgreSQL parameters, and execute one page for the current test user.

Verify three result classes when data exists:

```text
example_source = word
example_source = best_sentence
example_source = none
```

Also verify:

- returned row count does not exceed the requested page size;
- total count remains the unfinished unique-word count;
- no duplicate `progress_key` appears;
- keyword filtering still checks the wrong word only.

- [x] **Step 2: Run the complete Java suite**

Run:

```bash
cd rob_english_word_back
JAVA_HOME=/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home \
  mvn test
```

Expected: all Java tests PASS. If the sandbox blocks a test-only localhost port, rerun the identical command with host permission rather than changing production code.

- [x] **Step 3: Run the complete user-frontend suite and build**

Run:

```bash
cd rob_english_word_front
npm test
npm run build
```

Expected: all Vitest tests PASS and the production build succeeds.

- [x] **Step 4: Restart all services without Docker**

From the repository root:

```bash
./restart_all_services.sh restart
```

Expected: ports 6011 through 6018 are reported Ready. Keep the script defaults that do not start Docker dependencies or Docker stacks.

- [ ] **Step 5: Perform runtime API and visual checks**

Check:

```text
http://127.0.0.1:6011/wrong-words
```

Verify:

- the API response includes `exampleSentence` and `exampleSource`;
- original word-library examples win over AI-best examples;
- AI-best examples appear when the original sentence is blank;
- missing examples display `—`;
- the target word or phrase is highlighted without highlighting inner substrings;
- hovering a truncated example exposes the full sentence;
- pagination and sorting still work;
- at widths below 1280px, rows switch to cards without horizontal page overflow.

Runtime note: the production SQL was executed against the live database and all
service health checks passed. The isolated browser session redirected to `/login`;
the repository-local credential was not the application user's password. No
authentication bypass or password reset was attempted, so the signed-in API and
visual checks remain explicitly unverified.

- [x] **Step 6: Inspect fresh service logs and the final diff**

Run:

```bash
git diff --check
git status --short
```

Inspect only the post-restart time window for Java `ERROR` and frontend build/runtime errors. Confirm that pre-existing deployment-script and planning-file modifications remain untouched.

- [x] **Step 7: Request code review and fix any Critical or Important finding**

Review the feature commits against design commit `6dd2e94`. Re-run the focused tests after each review fix, then repeat Steps 2, 3, 5, and 6 before declaring completion.
