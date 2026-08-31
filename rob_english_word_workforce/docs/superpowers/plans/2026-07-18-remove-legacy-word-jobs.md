# Remove Legacy Word Jobs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the legacy sentence-generation, sentence-scoring, and TTS batch-job UI/API/worker code and delete their three PostgreSQL task tables while preserving all non-job `word-agent` capabilities and generated business data.

**Architecture:** Retire the feature from the outside in: remove React entry points, remove Go management/proxy endpoints, then remove only the three Python batch-job implementations. Prove each removal with a temporary failing absence test, retain the generic sentence/scoring/TTS paths, then delete the three task tables with an explicit database-name guard and no `CASCADE`.

**Tech Stack:** React 19, TypeScript 5, Vite 7, Go 1.25, Gin, GORM, Python 3.12, FastAPI, pytest, PostgreSQL/psycopg, Bash/Git.

---

## File map

- `word_select_dashboard/web-react/src/App.tsx`: remove the three menu entries, page keys, page components, task-only constants/imports, and render branches.
- `word_select_dashboard/web-react/src/lib/wordLibraryApi.ts`: retain normal word-library and single-score calls; remove legacy list/run/preview/import clients.
- `word_select_dashboard/web-react/src/types/wordLibrary.ts`: retain sentence/result models; remove legacy job, preview/import, and run-response models.
- `word_select_dashboard/web-react/src/styles/app.css`: remove task-page-only filter and table rules.
- `word_select_dashboard/server/router/system/sys_word_library.go`: expose only active word-library routes.
- `word_select_dashboard/server/api/v1/system/sys_word_library.go`: retain normal word-library and single-score handlers; remove legacy job handlers, preview/import logic, table initializers, query helpers, and three Python run proxies.
- `word_select_dashboard/word-agent/src/word_agent/api/routes.py`: retain health, generic run, sentence, single TTS, wrong-word, and single-score routes; remove three batch-job routes.
- `word_select_dashboard/word-agent/src/word_agent/domain/schemas.py`: remove the six batch-job request/response schemas only.
- `word_select_dashboard/word-agent/src/word_agent/services/`: delete three job services and the MinIO helper used only by the deleted TTS job.
- `word_select_dashboard/word-agent/pyproject.toml`, `uv.lock`, `.env.example`, and `core/config.py`: remove the now-unused Python MinIO dependency/settings.
- `rob_english_word_back/db/word_clean_sentence_job.sql`, `word_clean_sentence_score_job.sql`, and `word_clean_sentence_tts_job.sql`: delete obsolete DDL files.
- `DATABASES.md`: remove the three task-table and retired workflow descriptions while preserving current service and database guidance.

### Task 1: Remove the three React task-management pages

**Files:**
- Create temporarily, then delete: `word_select_dashboard/web-react/scripts/check_no_legacy_word_jobs.mjs`
- Modify: `word_select_dashboard/web-react/src/App.tsx`
- Modify: `word_select_dashboard/web-react/src/lib/wordLibraryApi.ts`
- Modify: `word_select_dashboard/web-react/src/types/wordLibrary.ts`
- Modify: `word_select_dashboard/web-react/src/styles/app.css`

- [ ] **Step 1: Write a temporary failing frontend absence test**

Create `word_select_dashboard/web-react/scripts/check_no_legacy_word_jobs.mjs` with:

```js
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const checks = new Map([
  [
    "src/App.tsx",
    [
      "造句任务表",
      "评分任务表",
      "TTS 任务表",
      "word-clean-jobs",
      "word-clean-score-jobs",
      "word-clean-tts-jobs",
      "WordCleanSentenceJobPage",
      "WordCleanSentenceScoreJobPage",
      "WordCleanSentenceTTSJobPage",
    ],
  ],
  [
    "src/lib/wordLibraryApi.ts",
    [
      "listWordCleanSentenceJobs",
      "listWordCleanSentenceScoreJobs",
      "listWordCleanSentenceTTSJobs",
      "previewWordCleanSentenceScorePayload",
      "importWordCleanSentenceScores",
      "startWordCleanSentenceJobs",
      "startWordCleanSentenceScoreJobs",
      "startWordCleanSentenceTTSJobs",
    ],
  ],
  [
    "src/types/wordLibrary.ts",
    [
      "WordCleanSentenceJobItem",
      "WordCleanSentenceScoreJobItem",
      "WordCleanSentenceScorePreview",
      "ImportWordCleanSentenceScoresResponse",
      "WordCleanSentenceTTSJobItem",
      "StartWordCleanSentenceJobsResponse",
      "StartWordCleanSentenceScoreJobsResponse",
      "StartWordCleanSentenceTTSJobsResponse",
    ],
  ],
  [
    "src/styles/app.css",
    [
      ".job-filter-bar",
      ".job-run-model-select",
      ".word-job-table",
      ".word-score-job-table",
      ".word-tts-job-table",
    ],
  ],
]);

const failures = [];
for (const [relativePath, forbiddenValues] of checks) {
  const source = readFileSync(resolve(root, relativePath), "utf8");
  for (const value of forbiddenValues) {
    if (source.includes(value)) failures.push(`${relativePath}: ${value}`);
  }
}

if (failures.length > 0) {
  console.error(failures.join("\n"));
  process.exit(1);
}
console.log("legacy word-job frontend code is absent");
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
node word_select_dashboard/web-react/scripts/check_no_legacy_word_jobs.mjs
```

Expected: exit `1` with matches from all four frontend files, including `App.tsx: 造句任务表`.

- [ ] **Step 3: Remove the legacy UI and client surface**

In `App.tsx`:

- Remove `FileTextOutlined` and `UploadOutlined`; retain `CloudSyncOutlined` and `HistoryOutlined` because active pages still use them.
- Remove the eight legacy API imports and four legacy type imports listed in Step 1.
- Remove `word-clean-jobs`, `word-clean-score-jobs`, and `word-clean-tts-jobs` from `PageKey`.
- Remove `WORD_CLEAN_SENTENCE_MODELS`, `WORD_CLEAN_SCORE_MODELS`, `WORD_CLEAN_TTS_PROVIDERS`, `WORD_CLEAN_TTS_MODELS`, `WORD_CLEAN_TTS_VOICES`, `WORD_CLEAN_JOB_STATUSES`, and `const { TextArea } = Input`.
- Delete the complete `WordCleanSentenceJobPage`, `WordCleanSentenceScoreJobPage`, and `WordCleanSentenceTTSJobPage` functions.
- Delete their three `renderPage()` branches and their three sidebar buttons.

In `wordLibraryApi.ts`, remove the eight functions listed in Step 1 and their task-only imported types. Keep this active call unchanged:

```ts
export function scoreWordCleanSentences(params: {
  ids?: number[];
  wordCleanIds?: number[];
  modelNames?: string[];
  judgeModel?: string;
  limit?: number;
  overwrite?: boolean;
}): Promise<ScoreWordCleanSentencesResponse> {
  return requestJSON<ScoreWordCleanSentencesResponse>("/word-libraries/clean-sentences/score", {
    method: "POST",
    body: JSON.stringify(params),
  });
}
```

In `types/wordLibrary.ts`, delete the eight task-only exported interfaces listed in Step 1 and retain `WordCleanSentenceItem`, `WordCleanItem`, and `ScoreWordCleanSentencesResponse`.

In `app.css`, delete the `.job-filter-bar` and `.job-run-model-select` blocks and every rule from `.word-job-table` through the final `.word-job-table` column-width rule. Do not remove shared `.word-library-table` or `.word-clean-table` rules.

- [ ] **Step 4: Verify GREEN and the production build**

Run:

```bash
node word_select_dashboard/web-react/scripts/check_no_legacy_word_jobs.mjs
npm run build --prefix word_select_dashboard/web-react
```

Expected: the absence test prints `legacy word-job frontend code is absent`; TypeScript and Vite exit `0`.

- [ ] **Step 5: Delete the temporary test and commit the frontend removal**

Delete `word_select_dashboard/web-react/scripts/check_no_legacy_word_jobs.mjs` with `apply_patch`, then run:

```bash
git add word_select_dashboard/web-react/src/App.tsx word_select_dashboard/web-react/src/lib/wordLibraryApi.ts word_select_dashboard/web-react/src/types/wordLibrary.ts word_select_dashboard/web-react/src/styles/app.css
git commit -m "refactor: 删除旧任务管理页面"
```

Expected: one commit containing only the four production frontend files.

### Task 2: Remove the legacy Go routes and handlers

**Files:**
- Create temporarily, then delete: `word_select_dashboard/server/router/system/sys_word_library_legacy_routes_test.go`
- Modify: `word_select_dashboard/server/router/system/sys_word_library.go`
- Modify: `word_select_dashboard/server/api/v1/system/sys_word_library.go`
- Delete: `word_select_dashboard/server/api/v1/system/sys_word_library_preview_test.go`
- Delete: `word_select_dashboard/server/api/v1/system/sys_word_library_import_scores_test.go`

- [ ] **Step 1: Write a temporary failing Go route-absence test**

Create `word_select_dashboard/server/router/system/sys_word_library_legacy_routes_test.go`:

```go
package system

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWordLibraryRouterDoesNotExposeLegacyJobRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&WordLibraryRouter{}).InitWordLibraryRouter(engine.Group("/api"))

	forbidden := map[string]struct{}{
		"/api/word-libraries/clean-sentences/scores/import":    {},
		"/api/word-libraries/clean-sentence-jobs":              {},
		"/api/word-libraries/clean-sentence-jobs/run":          {},
		"/api/word-libraries/clean-sentence-score-jobs":        {},
		"/api/word-libraries/clean-sentence-score-jobs/preview": {},
		"/api/word-libraries/clean-sentence-score-jobs/run":    {},
		"/api/word-libraries/clean-sentence-tts-jobs":          {},
		"/api/word-libraries/clean-sentence-tts-jobs/run":      {},
	}

	for _, route := range engine.Routes() {
		if _, exists := forbidden[route.Path]; exists {
			t.Errorf("legacy route is still registered: %s %s", route.Method, route.Path)
		}
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
cd word_select_dashboard/server && go test ./router/system -run TestWordLibraryRouterDoesNotExposeLegacyJobRoutes -v
```

Expected: FAIL with eight `legacy route is still registered` errors.

- [ ] **Step 3: Remove route registration and task-only Go code**

Reduce the route block to the active surface:

```go
{
	wordLibraryRouter.GET("", wordLibraryApi.Libraries)
	wordLibraryRouter.GET("clean-words", wordLibraryApi.CleanWords)
	wordLibraryRouter.GET("clean-words/:id/sentences", wordLibraryApi.CleanWordSentences)
	wordLibraryRouter.POST("clean-sentences/score", wordLibraryApi.ScoreCleanSentences)
	wordLibraryRouter.GET(":id/words", wordLibraryApi.Words)
}
```

From `sys_word_library.go`, delete:

- Task/preview/import types from `WordCleanSentenceJobItem` through `StartWordCleanSentenceTTSJobsResponse`, plus `ImportWordCleanSentenceScoresRequest`, `ImportWordCleanSentenceScoreItem`, `normalizedImportedScore`, `scoreJobImportDetails`, and `ImportWordCleanSentenceScoresResponse`.
- `ImportCleanSentenceScores`, `normalizeImportedScores`, `saveImportedScores`, `refreshBestSentencesForWordCleanIDs`, `refreshScoreJobsForImportedWordCleanIDs`, and `buildScoreJobStatus`.
- `CleanSentenceJobs`, `StartCleanSentenceJobs`, `CleanSentenceScoreJobs`, `CleanSentenceScorePreview`, `prepareScorePreviewJobs`, `selectScorePreviewMode`, `fetchScorePreviewRows`, `buildScorePreview`, `StartCleanSentenceScoreJobs`, `CleanSentenceTTSJobs`, and `StartCleanSentenceTTSJobs`.
- `callWordAgentStartCleanSentenceJobs`, `callWordAgentStartCleanSentenceScoreJobs`, and `callWordAgentStartCleanSentenceTTSJobs`; retain `callWordAgentScoreCleanSentences`.
- `ensureWordCleanSentenceScoreJobTable`, `ensureWordCleanSentenceTTSJobTable`, `wordCleanSentenceJobWhere`, `wordCleanSentenceScoreJobWhere`, and `wordCleanSentenceTTSJobWhere`.
- The now-unused `sort` import. Retain `gorm.io/gorm`, `ensureWordCleanSentenceScoreColumns`, and `ensureWordCleanBestSentenceTable`, which active word-clean endpoints still use.

Delete the two obsolete API test files listed above.

- [ ] **Step 4: Format and verify GREEN**

Run:

```bash
gofmt -w word_select_dashboard/server/router/system/sys_word_library.go word_select_dashboard/server/api/v1/system/sys_word_library.go word_select_dashboard/server/router/system/sys_word_library_legacy_routes_test.go
cd word_select_dashboard/server && go test ./router/system -run TestWordLibraryRouterDoesNotExposeLegacyJobRoutes -v && go test ./...
```

Expected: the route-absence test and the full Go suite pass with exit `0`.

- [ ] **Step 5: Delete the temporary test and commit the Go removal**

Delete `sys_word_library_legacy_routes_test.go` with `apply_patch`, rerun `go test ./...`, then:

```bash
git add word_select_dashboard/server/router/system/sys_word_library.go word_select_dashboard/server/api/v1/system/sys_word_library.go word_select_dashboard/server/api/v1/system/sys_word_library_preview_test.go word_select_dashboard/server/api/v1/system/sys_word_library_import_scores_test.go
git commit -m "refactor: 删除旧任务管理接口"
```

Expected: Go tests pass and the commit records the route/API deletion.

### Task 3: Remove only the three Python batch-job implementations

**Files:**
- Modify temporarily, then restore: `word_select_dashboard/word-agent/tests/test_api.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/api/routes.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/domain/schemas.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/core/config.py`
- Modify: `word_select_dashboard/word-agent/.env.example`
- Modify: `word_select_dashboard/word-agent/pyproject.toml`
- Modify: `word_select_dashboard/word-agent/uv.lock`
- Delete: `word_select_dashboard/word-agent/src/word_agent/services/word_clean_sentence_job.py`
- Delete: `word_select_dashboard/word-agent/src/word_agent/services/word_clean_sentence_score_job.py`
- Delete: `word_select_dashboard/word-agent/src/word_agent/services/word_clean_sentence_tts_job.py`
- Delete: `word_select_dashboard/word-agent/src/word_agent/services/minio_storage.py`
- Delete: `word_select_dashboard/word-agent/tests/test_word_clean_sentence_tts_job.py`
- Delete: `word_select_dashboard/word-agent/tests/test_minio_storage.py`
- Delete: `word_select_dashboard/word-agent/scripts/run_qwen_failed_rounds.py`
- Delete: `word_select_dashboard/word-agent/scripts/start_qwen_failed_rounds.py`

- [ ] **Step 1: Add a failing API-absence test**

Add `import pytest` and this test to `tests/test_api.py`:

```python
@pytest.mark.parametrize(
    "path",
    [
        "/v1/word-clean-sentences/run",
        "/v1/word-clean-sentence-score-jobs/run",
        "/v1/word-clean-sentence-tts-jobs/run",
    ],
)
def test_legacy_batch_job_routes_are_absent(path: str) -> None:
    client = TestClient(create_app())

    response = client.post(path, json={})

    assert response.status_code == 404
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
cd word_select_dashboard/word-agent && uv run pytest tests/test_api.py::test_legacy_batch_job_routes_are_absent -v
```

Expected: three failures because the current routes return validation responses instead of `404`.

- [ ] **Step 3: Remove the batch-job routes, schemas, services, and job-only MinIO code**

In `routes.py`:

- Remove `logging` and `threading` if they become unused.
- Remove the six job request/response imports and the three job service imports.
- Remove `_word_clean_sentence_run_lock`, `_word_clean_sentence_score_run_lock`, and `_word_clean_sentence_tts_run_lock`.
- Delete the three job route functions and three `_run_*` background helpers between the wrong-word endpoint and the retained single-score endpoint.
- Keep this active route and its service:

```python
@router.post("/v1/word-clean-sentences/score", response_model=WordCleanSentenceScoreResponse)
async def score_word_clean_sentences(
    request: WordCleanSentenceScoreRequest,
    settings: Annotated[Settings, Depends(get_settings)],
) -> WordCleanSentenceScoreResponse:
```

In `schemas.py`, delete `WordCleanSentenceRunRequest`, `WordCleanSentenceRunResponse`, `WordCleanSentenceScoreJobRunRequest`, `WordCleanSentenceScoreJobRunResponse`, `WordCleanSentenceTTSJobRunRequest`, and `WordCleanSentenceTTSJobRunResponse`. Retain `WordCleanSentenceScoreRequest` and all generic TTS schemas.

Delete the eight job-only files listed in the task file map. In `core/config.py`, remove the six `minio_*` fields only. In `.env.example`, remove the six `WORD_AGENT_MINIO_*` entries. In `pyproject.toml`, remove `"minio>=7.2.0"`.

- [ ] **Step 4: Refresh dependencies and verify GREEN**

Run:

```bash
cd word_select_dashboard/word-agent && uv lock
cd word_select_dashboard/word-agent && uv run pytest tests/test_api.py::test_legacy_batch_job_routes_are_absent -v
cd word_select_dashboard/word-agent && uv run pytest -v
cd word_select_dashboard/word-agent && uv run ruff check src tests
```

Expected: the temporary route test reports three passes; the remaining Python tests and Ruff pass; `uv.lock` no longer contains the `minio` package.

- [ ] **Step 5: Remove the temporary test and commit the Python removal**

Remove `import pytest` if no other test uses it and delete `test_legacy_batch_job_routes_are_absent`. Rerun `uv run pytest -v` and `uv run ruff check src tests`, then:

```bash
git add word_select_dashboard/word-agent
git commit -m "refactor: 删除旧批处理任务执行器"
```

Expected: the commit deletes only batch-job/MinIO-only code and preserves all generic API tests.

### Task 4: Remove obsolete DDL/docs and delete the live task tables

**Files:**
- Delete: `rob_english_word_back/db/word_clean_sentence_job.sql`
- Delete: `rob_english_word_back/db/word_clean_sentence_score_job.sql`
- Delete: `rob_english_word_back/db/word_clean_sentence_tts_job.sql`
- Modify: `DATABASES.md`

- [ ] **Step 1: Remove obsolete schema sources and current documentation**

Delete the three DDL files with `apply_patch`. In `DATABASES.md`, remove the `word_clean_sentence_job` inventory entry and the entire ``## `word_clean` Sentence Generation`` section. Keep the `word_clean_sentence` business-table entry, service restart rules, and database read instructions.

- [ ] **Step 2: Verify runtime and current-doc references are gone**

Run:

```bash
git grep -n -E '造句任务表|评分任务表|TTS 任务表|word_clean_sentence_(job|score_job|tts_job)|word-clean-sentence-(score-jobs|tts-jobs)|word-clean-sentences/run|clean-sentence-(jobs|score-jobs|tts-jobs)' -- . ':!docs/superpowers/**'
```

Expected: exit `1` with no output. Any match outside `docs/superpowers` must be removed before continuing.

- [ ] **Step 3: Read and guard the target database before deletion**

From `word_select_dashboard/word-agent`, run this read-only preflight using the retained single-score service to resolve the configured DSN without printing credentials:

```bash
uv run python -c '
import psycopg
from word_agent.core.config import get_settings
from word_agent.services.word_clean_sentence_score import WordCleanSentenceScoreService
s = get_settings()
dsn = WordCleanSentenceScoreService(s)._resolve_dsn()
with psycopg.connect(dsn) as conn:
    database = conn.execute("SELECT current_database()").fetchone()[0]
    assert database == "rob_english_word", f"refusing unexpected database: {database}"
    names = ["word_clean_sentence_job", "word_clean_sentence_score_job", "word_clean_sentence_tts_job"]
    for name in names:
        exists = conn.execute("SELECT to_regclass(%s)", (f"public.{name}",)).fetchone()[0]
        count = conn.execute(f"SELECT count(*) FROM public.{name}").fetchone()[0] if exists else 0
        print(name, "exists=" + str(bool(exists)), "rows=" + str(count))
'
```

Expected: database guard passes and the command prints existence/row counts for exactly the three target tables. If sandbox networking blocks localhost, rerun this exact command with approval outside the sandbox.

- [ ] **Step 4: Drop only the three guarded tables and prove business rows are unchanged**

Run:

```bash
uv run python -c '
import psycopg
from word_agent.core.config import get_settings
from word_agent.services.word_clean_sentence_score import WordCleanSentenceScoreService
s = get_settings()
dsn = WordCleanSentenceScoreService(s)._resolve_dsn()
with psycopg.connect(dsn) as conn:
    database = conn.execute("SELECT current_database()").fetchone()[0]
    assert database == "rob_english_word", f"refusing unexpected database: {database}"
    business_tables = ["word_clean_sentence", "word_clean_best_sentence"]
    before = {name: conn.execute(f"SELECT count(*) FROM public.{name}").fetchone()[0] for name in business_tables}
    conn.execute("DROP TABLE IF EXISTS public.word_clean_sentence_tts_job")
    conn.execute("DROP TABLE IF EXISTS public.word_clean_sentence_score_job")
    conn.execute("DROP TABLE IF EXISTS public.word_clean_sentence_job")
    after = {name: conn.execute(f"SELECT count(*) FROM public.{name}").fetchone()[0] for name in business_tables}
    assert after == before, (before, after)
    print("preserved business rows", after)
'
```

Expected: exit `0`, no `CASCADE` is used, and the printed before/after business counts are identical.

- [ ] **Step 5: Verify the tables are absent**

Run:

```bash
uv run python -c '
import psycopg
from word_agent.core.config import get_settings
from word_agent.services.word_clean_sentence_score import WordCleanSentenceScoreService
s = get_settings()
dsn = WordCleanSentenceScoreService(s)._resolve_dsn()
with psycopg.connect(dsn) as conn:
    database = conn.execute("SELECT current_database()").fetchone()[0]
    assert database == "rob_english_word", f"unexpected database: {database}"
    names = ["word_clean_sentence_job", "word_clean_sentence_score_job", "word_clean_sentence_tts_job"]
    remaining = [name for name in names if conn.execute("SELECT to_regclass(%s)", (f"public.{name}",)).fetchone()[0]]
    assert remaining == [], remaining
    print("legacy task tables are absent")
'
```

Expected: `legacy task tables are absent`.

- [ ] **Step 6: Commit schema and documentation cleanup**

```bash
git add DATABASES.md rob_english_word_back/db/word_clean_sentence_job.sql rob_english_word_back/db/word_clean_sentence_score_job.sql rob_english_word_back/db/word_clean_sentence_tts_job.sql
git commit -m "chore: 删除旧任务表定义"
```

Expected: one commit with three deleted DDL files and updated current documentation.

### Task 5: Full verification and service restart

**Files:**
- Verify only; no expected file changes.

- [ ] **Step 1: Run all static and test checks fresh**

Run:

```bash
npm run build --prefix word_select_dashboard/web-react
cd word_select_dashboard/server && go test ./...
cd word_select_dashboard/word-agent && uv run pytest -v && uv run ruff check src tests
git diff --check
```

Expected: every command exits `0`; no test failures, Ruff errors, TypeScript errors, Vite errors, or whitespace errors.

- [ ] **Step 2: Run the final residual-reference audit**

Run:

```bash
git grep -n -E '造句任务表|评分任务表|TTS 任务表|word_clean_sentence_(job|score_job|tts_job)|word-clean-sentence-(score-jobs|tts-jobs)|word-clean-sentences/run|clean-sentence-(jobs|score-jobs|tts-jobs)' -- . ':!docs/superpowers/**'
```

Expected: exit `1` with no output.

- [ ] **Step 3: Restart only the two changed backend services**

Run:

```bash
./word_select_dashboard/word-agent/restart_word_agent.sh
./word_select_dashboard/server/restart_word_select_dashboard_server.sh
```

Expected: `word_agent` becomes ready on `8010` and `word_select_dashboard_server` becomes ready on `8009`. If launchctl or port operations are blocked by the sandbox, rerun the exact failed command with approval outside the sandbox.

- [ ] **Step 4: Verify retained and removed HTTP behavior**

Run:

```bash
curl -fsS http://127.0.0.1:8010/health
curl -sS -o /dev/null -w '%{http_code}\n' -X POST -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8010/v1/word-clean-sentences/run
curl -sS -o /dev/null -w '%{http_code}\n' -X POST -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8010/v1/word-clean-sentence-score-jobs/run
curl -sS -o /dev/null -w '%{http_code}\n' -X POST -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8010/v1/word-clean-sentence-tts-jobs/run
```

Expected: health returns JSON with `"status":"ok"`; each removed route prints `404`. Do not invoke retained generation/scoring/TTS endpoints during verification because they can spend model credits or write business data.

- [ ] **Step 5: Verify repository state and report the destructive action**

Run:

```bash
git status --short --branch
git log -5 --oneline
```

Expected: no uncommitted implementation changes. Report that the three PostgreSQL task tables and their history were permanently deleted, that generated business data was preserved, and list the fresh build/test/restart evidence.
