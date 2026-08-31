# Wrong Word Event-Driven Retry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace synchronous poison-batch processing with a PostgreSQL-backed processor that retries failed three-word batches only when a newer wrong-word event arrives or the service starts, while preserving Java-side sentence/TTS generation idempotency.

**Architecture:** `WrongWordStrategyService` becomes a synchronous PostgreSQL queue service with separate enqueue and drain operations. A new application-scoped `WrongWordEventProcessor` owns one `asyncio.Event`, wakes on committed inserts and once at startup, and runs blocking queue work through `asyncio.to_thread`. Java accepts a stable `generationKey`, returns an existing `sentence_cloze_item` before generating again, and protects the final insert with a unique index.

**Tech Stack:** Python 3.12, FastAPI lifespan, asyncio, psycopg 3, httpx, pytest; Java 21, Spring Boot 3.2, MyBatis-Plus, PostgreSQL, JUnit 5, Mockito.

---

## File map

- Create `word_select_dashboard/word-agent/src/word_agent/services/wrong_word_processor.py`: application-scoped wake/stop lifecycle only; no SQL.
- Modify `word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py`: schema upgrades, enqueue, row claiming, retry watermarks, failure isolation, stable generation keys.
- Modify `word_select_dashboard/word-agent/src/word_agent/main.py`: start and stop the processor through FastAPI lifespan.
- Modify `word_select_dashboard/word-agent/src/word_agent/api/routes.py`: enqueue synchronously in a worker thread, then notify the processor after commit.
- Modify `word_select_dashboard/word-agent/src/word_agent/core/config.py`: processor batch and hard-timeout settings.
- Create `word_select_dashboard/word-agent/tests/test_wrong_word_processor.py`: no-polling and wake lifecycle tests.
- Create `word_select_dashboard/word-agent/tests/test_wrong_word_strategy.py`: SQL/state transition tests using a recording fake connection plus focused batch-processing fakes.
- Modify `word_select_dashboard/word-agent/tests/test_api.py`: lightweight enqueue response and processor notification contract.
- Modify `rob_english_word_back/db/sentence_cloze_item.sql`: nullable `generation_key` plus partial unique index.
- Modify `rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeItem.java`: map `generationKey`.
- Modify `rob_english_word_back/src/main/java/com/robword/dto/SentenceClozeGenerateRequest.java`: accept `generationKey`.
- Modify `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`: retrieve an item by generation key.
- Modify `rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java`: return existing generated content before calling word-agent and recover unique-key races.
- Modify `rob_english_word_back/src/test/java/com/robword/service/SentenceClozeServiceTest.java`: prove repeat requests do not regenerate or insert.
- Modify `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java`: assert the generation-key lookup SQL contract.

### Task 1: Add Java generation-key persistence contract

**Files:**
- Modify: `rob_english_word_back/db/sentence_cloze_item.sql`
- Modify: `rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeItem.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/SentenceClozeGenerateRequest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`
- Test: `rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java`

- [ ] **Step 1: Write the failing mapper contract test**

```java
@Test
void generationKeyLookupUsesUniqueBusinessKey() throws Exception {
    Method method = SentenceClozeItemMapper.class.getMethod(
            "selectByGenerationKey", String.class
    );
    String sql = String.join("\n", method.getAnnotation(Select.class).value());

    assertTrue(sql.contains("generation_key = #{generationKey}"));
    assertTrue(sql.contains("LIMIT 1"));
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```bash
cd rob_english_word_back
mvn -q -Dtest=SentenceClozeItemMapperContractTest test
```

Expected: FAIL because `selectByGenerationKey(String)` does not exist.

- [ ] **Step 3: Add the schema, entity, request, and mapper fields**

Append the idempotent schema migration:

```sql
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS generation_key varchar(160) NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_sentence_cloze_item_generation_key
    ON public.sentence_cloze_item(generation_key)
    WHERE generation_key IS NOT NULL;
```

Add `private String generationKey;` to both `SentenceClozeItem` and `SentenceClozeGenerateRequest`. Add this mapper method:

```java
@Select("""
        SELECT *
        FROM sentence_cloze_item
        WHERE generation_key = #{generationKey}
        LIMIT 1
        """)
SentenceClozeItem selectByGenerationKey(@Param("generationKey") String generationKey);
```

- [ ] **Step 4: Run the mapper contract test and verify it passes**

Run:

```bash
cd rob_english_word_back
mvn -q -Dtest=SentenceClozeItemMapperContractTest test
```

Expected: PASS.

- [ ] **Step 5: Commit the persistence contract**

```bash
git add rob_english_word_back/db/sentence_cloze_item.sql \
  rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeItem.java \
  rob_english_word_back/src/main/java/com/robword/dto/SentenceClozeGenerateRequest.java \
  rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java \
  rob_english_word_back/src/test/java/com/robword/mapper/SentenceClozeItemMapperContractTest.java
git commit -m "feat: add cloze generation idempotency key"
```

### Task 2: Make Java sentence generation idempotent

**Files:**
- Modify: `rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java`
- Test: `rob_english_word_back/src/test/java/com/robword/service/SentenceClozeServiceTest.java`

- [ ] **Step 1: Write failing service tests for reuse and insertion**

Add a test that prepares a stored item, returns it from `selectByGenerationKey`, and proves neither the HTTP server nor `insert` is used:

```java
@Test
void returnsExistingItemForRepeatedGenerationKeyWithoutCallingWordAgent() {
    SentenceClozeItemMapper mapper = mock(SentenceClozeItemMapper.class);
    SentenceClozeItem existing = storedItem("wrong-word-events:47-48-49");
    when(mapper.selectByGenerationKey("wrong-word-events:47-48-49"))
            .thenReturn(existing);
    SentenceClozeService service = createServiceWithoutServer(mapper);
    SentenceClozeGenerateRequest request = request();
    request.setGenerationKey("wrong-word-events:47-48-49");

    SentenceClozeGenerateResponse response = service.generateAndSave(request);

    assertEquals(existing.getId(), response.getId());
    verify(mapper, never()).insert(any(SentenceClozeItem.class));
}
```

Extend the successful-generation test to assert:

```java
request.setGenerationKey("wrong-word-events:47-48-49");
assertEquals("wrong-word-events:47-48-49", captor.getValue().getGenerationKey());
```

- [ ] **Step 2: Run the service tests and verify they fail**

Run:

```bash
cd rob_english_word_back
mvn -q -Dtest=SentenceClozeServiceTest test
```

Expected: FAIL because the service neither queries nor persists `generationKey`.

- [ ] **Step 3: Implement lookup-before-generation and duplicate-key recovery**

Normalize blank keys to `null`, query before `callWordAgent`, and set the key on a new entity. Extract stored JSON conversion so both existing and new records use the same response path:

```java
String generationKey = normalizeGenerationKey(request.getGenerationKey());
if (generationKey != null) {
    SentenceClozeItem existing = sentenceClozeItemMapper.selectByGenerationKey(generationKey);
    if (existing != null) {
        return toStoredResponse(existing);
    }
}

// Generate sentence/TTS and populate the entity.
item.setGenerationKey(generationKey);
try {
    sentenceClozeItemMapper.insert(item);
} catch (DuplicateKeyException exception) {
    if (generationKey == null) {
        throw exception;
    }
    SentenceClozeItem existing = sentenceClozeItemMapper.selectByGenerationKey(generationKey);
    if (existing == null) {
        throw exception;
    }
    return toStoredResponse(existing);
}
```

`toStoredResponse` must parse `wordsJson` and all four source-ID JSON fields with `ObjectMapper`, and must populate the same fields currently populated by `toResponse`.

- [ ] **Step 4: Run Java unit tests**

Run:

```bash
cd rob_english_word_back
mvn -q -Dtest=SentenceClozeServiceTest,SentenceClozeItemMapperContractTest test
```

Expected: PASS with no extra insert for a repeated key.

- [ ] **Step 5: Commit Java idempotency behavior**

```bash
git add rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java \
  rob_english_word_back/src/test/java/com/robword/service/SentenceClozeServiceTest.java
git commit -m "feat: reuse generated cloze items by generation key"
```

### Task 3: Convert Python wrong-word persistence into a durable queue

**Files:**
- Modify: `word_select_dashboard/word-agent/src/word_agent/core/config.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py`
- Create: `word_select_dashboard/word-agent/tests/test_wrong_word_strategy.py`

- [ ] **Step 1: Write failing queue-state tests**

Use a recording fake connection/cursor to assert the schema upgrades and state transitions. Cover these exact outcomes:

```python
def test_schema_adds_retry_watermark_and_lock_columns():
    service, conn = service_with_recording_connection()
    service.ensure_schema()
    sql = "\n".join(conn.executed_sql)
    assert "retry_count" in sql
    assert "retry_after_event_id" in sql
    assert "locked_at" in sql
    assert "last_attempt_at" in sql


def test_failed_batch_waits_for_newer_event():
    service, conn = service_with_batch(ids=[47, 48, 49], max_event_id=53)
    service.mark_batch_failed(conn, [47, 48, 49], "HTTP 400")
    assert conn.last_params["retry_after_event_id"] == 53
    assert "status = 'retry_wait'" in conn.last_sql


def test_third_failure_is_isolated():
    service, conn = service_with_batch(ids=[47, 48, 49], retry_count=2)
    service.mark_batch_failed(conn, [47, 48, 49], "HTTP 400")
    assert "status = 'failed'" in conn.last_sql
```

Also assert that retry reservation requires a same-user event with `id > retry_after_event_id`, and that fresh reservation reads only `status='pending'`.

- [ ] **Step 2: Run the queue-state tests and verify they fail**

Run:

```bash
cd word_select_dashboard/word-agent
.venv/bin/pytest tests/test_wrong_word_strategy.py -q
```

Expected: FAIL because the new queue API and columns do not exist.

- [ ] **Step 3: Add idempotent schema upgrades and configuration**

Add settings:

```python
wrong_word_max_batches_per_wake: int = 10
wrong_word_max_retries: int = 3
wrong_word_processing_timeout_seconds: float = 600.0
```

Expose a public `ensure_schema()` and make `_ensure_table()` execute one idempotent `ALTER TABLE ADD COLUMN IF NOT EXISTS` statement per field:

```sql
retry_count int NOT NULL DEFAULT 0,
retry_after_event_id bigint NULL,
locked_at timestamp NULL,
last_attempt_at timestamp NULL
```

Add indexes on `(user_id, status, retry_after_event_id, id)` and `(status, locked_at)`. Add `_migrate_legacy_failed_batches(conn)`: select errored `pending` rows ordered by `user_id, id`, split each user's rows into groups of three, derive `wrong-word-events:<sorted ids>`, and update each full group to `retry_wait`, `retry_count=1`, and the user's current maximum event ID. Log and preserve an incomplete legacy group instead of mixing it into another user's batch.

- [ ] **Step 4: Separate enqueue from processing**

Replace `handle_event` with an enqueue-only method and keep a compatibility wrapper only if tests or callers require it:

```python
def enqueue_event(self, event: WrongWordEventRequest) -> WrongWordEventResponse:
    with self._connect() as conn:
        self._ensure_table(conn)
        event_id = self._insert_event(conn, event)
        pending_count = self._pending_count(conn, event.user_id)
    return WrongWordEventResponse(
        event_id=event_id,
        pending_count=pending_count,
        generated=False,
    )
```

The enqueue method must not call Java and must return after the insert transaction commits.

- [ ] **Step 5: Implement stable claiming and failure isolation**

Generate the batch key before the external call:

```python
def _batch_key(self, batch: list[PendingWrongWord]) -> str:
    return "wrong-word-events:" + "-".join(str(item.id) for item in batch)
```

Include it in Java's payload:

```python
payload = {
    "generationKey": batch_key,
    "userId": batch[0].user_id,
    "userName": user_name,
    "words": words,
    "sourceEventIds": [item.id for item in batch],
    "sourceAnswerDetailIds": [item.source_answer_detail_id for item in batch],
    "sourceRecordIds": self._compact_ints(item.record_id for item in batch),
    "sourceWordIds": self._compact_ints(item.word_id for item in batch),
}
```

Fresh claims select exactly three `pending` rows for one user. Retry claims select one stable `batch_key` whose rows are all `retry_wait`, `retry_count < max`, and whose user has a newer event than `retry_after_event_id`. Both claims use `FOR UPDATE SKIP LOCKED`, set `processing`, `locked_at`, and `last_attempt_at`, then commit before the HTTP call.

On failure, capture response URL/status/body when the exception is `httpx.HTTPStatusError`, truncate the stored error, increment `retry_count`, and execute one of:

```sql
-- retries remain
status = 'retry_wait',
retry_after_event_id = (SELECT MAX(id) FROM wrong_word_events WHERE user_id = :user_id)

-- retry limit reached
status = 'failed'
```

Continue draining after a failed batch instead of raising to the API caller.

- [ ] **Step 6: Implement startup recovery and hard timeout**

Add the return type and public recovery operations:

```python
@dataclass(frozen=True)
class DrainResult:
    processed_batches: int
    has_more: bool
    remaining_startup_batch_keys: frozenset[str] = frozenset()


def recover_stale_processing(self) -> None:
    with self._connect() as conn:
        self._recover_stale_processing(conn)


def snapshot_startup_retry_batch_keys(self) -> set[str]:
    with self._connect() as conn:
        self._ensure_table(conn)
        rows = conn.execute(
            """
            SELECT DISTINCT batch_key
            FROM public.wrong_word_events
            WHERE status = 'retry_wait'
              AND retry_count < %(max_retries)s
              AND batch_key IS NOT NULL
            """,
            {"max_retries": self.settings.wrong_word_max_retries},
        ).fetchall()
    return {str(row["batch_key"]) for row in rows}
```

`process_available(startup_batch_keys=startup_batch_keys)` copies the supplied set, claims one startup key at a time, then normally eligible retry batches, then fresh batches. After every claim it calls `_process_reserved_batch`, increments the processed count, and removes the attempted startup key. It returns `DrainResult` containing unattempted startup keys and `has_more=True` only when those keys or other work eligible in this same trigger remain. Wrap each external generation call with the configured hard timeout; a timeout follows the same retry/failed transition.

- [ ] **Step 7: Run Python queue tests and lint**

Run:

```bash
cd word_select_dashboard/word-agent
.venv/bin/pytest tests/test_wrong_word_strategy.py -q
.venv/bin/ruff check src/word_agent/services/wrong_word_strategy.py tests/test_wrong_word_strategy.py
```

Expected: all tests PASS and ruff reports no errors.

- [ ] **Step 8: Commit the durable queue behavior**

```bash
git add word_select_dashboard/word-agent/src/word_agent/core/config.py \
  word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py \
  word_select_dashboard/word-agent/tests/test_wrong_word_strategy.py
git commit -m "feat: isolate failed wrong-word batches"
```

### Task 4: Add the event-driven processor and FastAPI lifecycle

**Files:**
- Create: `word_select_dashboard/word-agent/src/word_agent/services/wrong_word_processor.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/main.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/api/routes.py`
- Create: `word_select_dashboard/word-agent/tests/test_wrong_word_processor.py`
- Modify: `word_select_dashboard/word-agent/tests/test_api.py`

- [ ] **Step 1: Write failing processor tests**

Use a fake strategy service with call counters:

```python
async def eventually(predicate) -> None:
    for _ in range(100):
        if predicate():
            return
        await asyncio.sleep(0.01)
    raise AssertionError("processor did not reach expected state")


def test_processor_runs_once_on_start_and_does_not_poll():
    async def scenario() -> None:
        service = FakeStrategyService()
        processor = WrongWordEventProcessor(service)
        await processor.start()
        await eventually(lambda: service.startup_calls == 1)
        await asyncio.sleep(0.05)
        assert service.normal_calls == 0
        await processor.stop()

    asyncio.run(scenario())


def test_new_event_wakes_processor_once():
    async def scenario() -> None:
        service = FakeStrategyService()
        processor = WrongWordEventProcessor(service)
        await processor.start()
        await eventually(lambda: service.startup_calls == 1)
        processor.notify()
        await eventually(lambda: service.normal_calls == 1)
        await processor.stop()

    asyncio.run(scenario())
```

Add an API test whose fake strategy exposes `enqueue_event` and whose fake processor records `notify()`.

- [ ] **Step 2: Run the focused tests and verify they fail**

Run:

```bash
cd word_select_dashboard/word-agent
.venv/bin/pytest tests/test_wrong_word_processor.py tests/test_api.py::test_receive_wrong_word_event_uses_strategy_service -q
```

Expected: FAIL because `WrongWordEventProcessor` and enqueue/notify wiring do not exist.

- [ ] **Step 3: Implement the processor without a periodic timer**

Create the processor with this public interface and lifecycle:

```python
class WrongWordEventProcessor:
    def __init__(self, strategy: WrongWordStrategyService) -> None:
        self._strategy = strategy
        self._wake_event = asyncio.Event()
        self._stopping = False
        self._task: asyncio.Task[None] | None = None
        self._startup_batch_keys: set[str] = set()

    async def start(self) -> None:
        await asyncio.to_thread(self._strategy.ensure_schema)
        await asyncio.to_thread(self._strategy.recover_stale_processing)
        self._startup_batch_keys = await asyncio.to_thread(
            self._strategy.snapshot_startup_retry_batch_keys
        )
        self._stopping = False
        self._task = asyncio.create_task(self._run())
        self._wake_event.set()

    def notify(self) -> None:
        self._wake_event.set()

    async def stop(self) -> None:
        self._stopping = True
        self._wake_event.set()
        if self._task is not None:
            await self._task
            self._task = None

    async def _run(self) -> None:
        while True:
            await self._wake_event.wait()
            self._wake_event.clear()
            if self._stopping:
                return
            result = await asyncio.to_thread(
                self._strategy.process_available,
                startup_batch_keys=self._startup_batch_keys,
            )
            self._startup_batch_keys = set(result.remaining_startup_batch_keys)
            if result.has_more or self._startup_batch_keys:
                self._wake_event.set()
```

`start()` snapshots startup retry keys before the first drain. `notify()` only sets the event. `stop()` sets the stopping flag and event, then awaits the task. No `sleep`, loop interval, scheduler, or `next_retry_at` logic is allowed.

- [ ] **Step 4: Wire FastAPI lifespan and route notification**

Use `@asynccontextmanager` in `main.py`:

```python
@asynccontextmanager
async def lifespan(app: FastAPI):
    strategy = WrongWordStrategyService(get_settings())
    processor = WrongWordEventProcessor(strategy)
    app.state.wrong_word_strategy = strategy
    app.state.wrong_word_processor = processor
    await processor.start()
    try:
        yield
    finally:
        await processor.stop()
```

The endpoint uses the shared strategy and notifies only after `enqueue_event` returns:

```python
response = await asyncio.to_thread(
    request.app.state.wrong_word_strategy.enqueue_event,
    request_body,
)
request.app.state.wrong_word_processor.notify()
return response
```

Keep persistence failures as HTTP 500. Processing failures are stored by the processor and must never change the already returned enqueue response to HTTP 502.

- [ ] **Step 5: Run processor, API, and full Python tests**

Run:

```bash
cd word_select_dashboard/word-agent
.venv/bin/pytest tests/test_wrong_word_processor.py tests/test_api.py tests/test_wrong_word_strategy.py -q
.venv/bin/pytest -q
.venv/bin/ruff check src tests
```

Expected: all tests PASS, and waiting without notifications does not increase the fake service call count.

- [ ] **Step 6: Commit event-driven lifecycle wiring**

```bash
git add word_select_dashboard/word-agent/src/word_agent/services/wrong_word_processor.py \
  word_select_dashboard/word-agent/src/word_agent/main.py \
  word_select_dashboard/word-agent/src/word_agent/api/routes.py \
  word_select_dashboard/word-agent/tests/test_wrong_word_processor.py \
  word_select_dashboard/word-agent/tests/test_api.py
git commit -m "feat: wake wrong-word processing on new events"
```

### Task 5: Apply schema upgrades and verify the real queue

**Files:**
- Verify: `word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py`
- Verify: `rob_english_word_back/db/sentence_cloze_item.sql`

- [ ] **Step 1: Run the Java schema script against the configured database**

Use the PostgreSQL connection documented by the project and execute:

```bash
psql -h 127.0.0.1 -p 5432 -U conchi -d rob_english_word \
  -v ON_ERROR_STOP=1 -f rob_english_word_back/db/sentence_cloze_item.sql
```

Expected: `generation_key` exists and `uk_sentence_cloze_item_generation_key` is present.

- [ ] **Step 2: Trigger the Python idempotent schema upgrade**

Start word-agent once or call `WrongWordStrategyService.ensure_schema()` with the configured select database DSN.

Expected columns in `wrong_word_events`: `retry_count`, `retry_after_event_id`, `locked_at`, `last_attempt_at`.

- [ ] **Step 3: Verify existing queue data was preserved**

Run a read-only query grouped by status:

```sql
SELECT status, COUNT(*)
FROM public.wrong_word_events
GROUP BY status
ORDER BY status;
```

Expected: no records were deleted. Previously errored rows are either isolated in `retry_wait`/`failed` after startup processing, and later normal rows are no longer blocked behind them.

- [ ] **Step 4: Verify new-event-triggered retry behavior**

Insert or produce one real new wrong answer through the existing 7002/7003 flow, then query the old batch and the new event.

Expected:

- the HTTP wrong-word notification returns without waiting for sentence/TTS generation;
- the old failed batch attempt count increases at most once;
- if it fails again, its `retry_after_event_id` becomes the newest event ID;
- fresh rows continue toward a three-word batch;
- no additional attempt occurs while the system remains idle.

- [ ] **Step 5: Verify Java idempotency with the same key twice**

POST the same valid payload and `generationKey` twice to `/api/external/sentence-cloze/generate`.

Expected: both responses contain the same item ID; the second request does not call sentence generation/TTS and the table contains exactly one row for that key.

### Task 6: Final regression and handoff

**Files:**
- Verify only; no planned source changes.

- [ ] **Step 1: Run all targeted backend checks**

```bash
cd word_select_dashboard/word-agent
.venv/bin/pytest -q
.venv/bin/ruff check src tests

cd ../../rob_english_word_back
mvn -q test
```

Expected: Python tests, ruff, and Maven tests all PASS.

- [ ] **Step 2: Inspect the final diff for scope**

```bash
git diff --check
git status --short
git diff -- word_select_dashboard/word-agent rob_english_word_back
```

Expected: no whitespace errors; unrelated pre-existing frontend and planning-file changes remain untouched.

- [ ] **Step 3: Record runtime evidence**

Capture the final queue status counts, the retried batch IDs and attempt counts, the generated `sentence_cloze_item.id`, its MinIO `sentence_audio_url`, and the duplicate-key verification result in the handoff response.
