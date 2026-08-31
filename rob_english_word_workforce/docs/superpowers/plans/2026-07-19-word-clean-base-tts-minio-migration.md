# Base Word TTS MinIO Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Safely migrate every successful `word_clean_tts` WAV from the task-center disk to MinIO, switch the database to proxy URLs, and delete only the fully verified local source files.

**Architecture:** A standalone, resumable migration command reads successful base-word rows, resolves each exact local WAV, uploads it under `word_clean_tts/<filename>`, verifies MinIO size and MD5, and conditionally updates the row. A separate full-set validation gate checks database fields, MinIO objects, local files, and proxy playback before an explicit deletion mode may unlink the DB-referenced files.

**Tech Stack:** Python 3.12, psycopg 3, MinIO Python SDK, pytest, PostgreSQL, MinIO.

---

## File map

- Create `word_select_dashboard/word-agent/scripts/migrate_word_clean_tts_to_minio.py`: migration, verification, proxy sampling, and guarded local deletion command.
- Create `word_select_dashboard/word-agent/tests/test_migrate_word_clean_tts_to_minio.py`: unit tests for source validation, idempotent upload, database guards, verification gate, and exact-file deletion.
- Reuse `word_select_dashboard/word-agent/src/word_agent/core/config.py`: database and MinIO configuration; no changes expected.

### Task 1: Define migration records and local-file validation

**Files:**
- Create: `word_select_dashboard/word-agent/tests/test_migrate_word_clean_tts_to_minio.py`
- Create: `word_select_dashboard/word-agent/scripts/migrate_word_clean_tts_to_minio.py`

- [ ] **Step 1: Write failing tests for exact source resolution and WAV validation**

Test a valid `word_clean_1_tts_1.wav`, missing file, path traversal filename, database-size mismatch, and invalid RIFF/WAVE header. Import the script with `importlib.util.spec_from_file_location` so the production script stays runnable without becoming an application package.

```python
def test_build_item_accepts_valid_wav(tmp_path):
    source = tmp_path / "word_clean_1_tts_1.wav"
    source.write_bytes(b"RIFF\x04\x00\x00\x00WAVEdata")
    item = migration.build_item(
        row=(1, 9, "word", "word_clean_1_tts_1.wav",
             "http://127.0.0.1:19186/api/tts/files/word_clean_1_tts_1.wav", 16),
        task_center_dir=tmp_path,
        bucket="ai-file-navigation",
    )
    assert item.tts_id == 1
    assert item.object_key == "word_clean_tts/word_clean_1_tts_1.wav"
    assert item.object_url == "/ai-file-navigation/word_clean_tts/word_clean_1_tts_1.wav"
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `.venv/bin/pytest tests/test_migrate_word_clean_tts_to_minio.py -q`

Expected: FAIL because the migration module and `build_item` do not exist.

- [ ] **Step 3: Implement argument parsing, immutable record types, and validation**

Implement `MigrationItem`, `ValidationSummary`, `parse_args()`, `rob_word_dsn()`, `load_minio_config()`, `read_wav_header()`, and `build_item()`. Required flags are `--dry-run`, `--limit`, `--workers`, `--batch-size`, `--task-center-dir`, `--verify-proxy-base-url`, `--sample-size`, and `--delete-local`. Reject non-basename keys, unsupported local URLs, empty filenames, duplicate filenames, missing files, size mismatches, and non-WAV content.

- [ ] **Step 4: Run tests and lint**

Run: `.venv/bin/pytest tests/test_migrate_word_clean_tts_to_minio.py -q`

Expected: PASS for the local-file cases.

Run: `.venv/bin/ruff check scripts/migrate_word_clean_tts_to_minio.py tests/test_migrate_word_clean_tts_to_minio.py`

Expected: `All checks passed!`

- [ ] **Step 5: Commit the validation foundation**

```bash
git add word_select_dashboard/word-agent/scripts/migrate_word_clean_tts_to_minio.py word_select_dashboard/word-agent/tests/test_migrate_word_clean_tts_to_minio.py
git commit -m "feat: validate base word tts migration sources"
```

### Task 2: Add resumable upload and guarded database updates

**Files:**
- Modify: `word_select_dashboard/word-agent/scripts/migrate_word_clean_tts_to_minio.py`
- Modify: `word_select_dashboard/word-agent/tests/test_migrate_word_clean_tts_to_minio.py`

- [ ] **Step 1: Write failing tests for idempotent upload and conditional updates**

Use fake MinIO and database connections to prove that matching objects are reused, missing or mismatched objects are uploaded, post-upload size/ETag mismatches fail, and an update affecting anything other than exactly one row fails.

```python
def test_upload_reuses_matching_minio_object(valid_item):
    client = FakeMinio(size=valid_item.file_size, etag=migration.file_md5(valid_item.source_path))
    migration.upload_and_verify(client, "ai-file-navigation", valid_item)
    assert client.uploads == []

def test_update_database_requires_one_guarded_row(valid_item):
    conn = FakeConnection(update_rowcount=0)
    with pytest.raises(RuntimeError, match="Database guard failed"):
        migration.update_database(conn, bucket="ai-file-navigation", items=[valid_item])
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `.venv/bin/pytest tests/test_migrate_word_clean_tts_to_minio.py -q`

Expected: FAIL because upload and database functions are absent.

- [ ] **Step 3: Implement fetching, MD5 upload verification, and guarded updates**

Fetch only `status = 'success'` rows whose URL still points to `19186`, ordered by ID. Upload with `audio/wav`, verify `stat_object.size` and normalized ETag against local MD5, then update `tts_bucket`, `tts_object_key`, `tts_object_url`, and `updated_at` with guards on ID, success status, old URL, object key, and file size. Commit updates in configurable batches. Already-migrated rows must be excluded from uploads and retained for full verification.

- [ ] **Step 4: Run tests and lint**

Run: `.venv/bin/pytest tests/test_migrate_word_clean_tts_to_minio.py -q`

Expected: PASS.

Run: `.venv/bin/ruff check scripts/migrate_word_clean_tts_to_minio.py tests/test_migrate_word_clean_tts_to_minio.py`

Expected: `All checks passed!`

- [ ] **Step 5: Commit resumable migration support**

```bash
git add word_select_dashboard/word-agent/scripts/migrate_word_clean_tts_to_minio.py word_select_dashboard/word-agent/tests/test_migrate_word_clean_tts_to_minio.py
git commit -m "feat: migrate base word tts to minio"
```

### Task 3: Add the full verification gate and exact-file deletion

**Files:**
- Modify: `word_select_dashboard/word-agent/scripts/migrate_word_clean_tts_to_minio.py`
- Modify: `word_select_dashboard/word-agent/tests/test_migrate_word_clean_tts_to_minio.py`

- [ ] **Step 1: Write failing tests for gate failures and exact deletion**

Cover wrong bucket/key/URL, legacy `19186` URL, missing MinIO object, DB/MinIO/local size mismatch, proxy non-200 or non-WAV response, partial runs attempting `--delete-local`, and a passing full set that unlinks only its explicit source paths.

```python
def test_delete_local_requires_complete_validation(valid_item):
    summary = migration.ValidationSummary(expected=1, verified=0, failures=("missing",))
    with pytest.raises(RuntimeError, match="refusing local deletion"):
        migration.delete_verified_sources([valid_item], summary)
    assert valid_item.source_path.exists()

def test_delete_local_unlinks_only_verified_items(valid_item, tmp_path):
    unrelated = tmp_path / "unrelated.wav"
    unrelated.write_bytes(b"RIFF\x04\x00\x00\x00WAVEdata")
    summary = migration.ValidationSummary(expected=1, verified=1, failures=())
    assert migration.delete_verified_sources([valid_item], summary) == 1
    assert not valid_item.source_path.exists()
    assert unrelated.exists()
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `.venv/bin/pytest tests/test_migrate_word_clean_tts_to_minio.py -q`

Expected: FAIL because the gate and deletion functions are absent.

- [ ] **Step 3: Implement full verification and proxy sampling**

Load every success row, require target bucket/key/URL fields, verify every MinIO object size, compare every still-present local source size, and sample at least 20 evenly distributed IDs through the configured proxy. A proxy sample passes only on HTTP 200 with an audio-compatible content type and a RIFF/WAVE header. Report structured counters for total, verified, field mismatch, missing object, size mismatch, local mismatch, and proxy failure.

- [ ] **Step 4: Implement the deletion gate**

Allow deletion only when no `--limit` is set, the total equals the database success count captured at runtime, all counters are zero, all success URLs no longer contain `19186`, and every exact source path is inside the resolved task-center directory. Call `Path.unlink()` per verified item, never remove the directory, and fail before the first unlink if any precondition fails.

- [ ] **Step 5: Run tests, lint, and the existing word-agent suite**

Run: `.venv/bin/pytest tests/test_migrate_word_clean_tts_to_minio.py -q`

Expected: PASS.

Run: `.venv/bin/pytest -q`

Expected: all word-agent tests PASS.

Run: `.venv/bin/ruff check scripts/migrate_word_clean_tts_to_minio.py tests/test_migrate_word_clean_tts_to_minio.py`

Expected: `All checks passed!`

- [ ] **Step 6: Commit guarded cleanup support**

```bash
git add word_select_dashboard/word-agent/scripts/migrate_word_clean_tts_to_minio.py word_select_dashboard/word-agent/tests/test_migrate_word_clean_tts_to_minio.py
git commit -m "feat: verify and clean migrated word tts"
```

### Task 4: Execute and independently verify the production migration

**Files:**
- No repository files expected.

- [ ] **Step 1: Run a read-only dry run**

Run from `word_select_dashboard/word-agent`:

```bash
.venv/bin/python scripts/migrate_word_clean_tts_to_minio.py --dry-run
```

Expected before upload: 21,888 migratable success rows, 210 pending rows, no missing or invalid local WAV files, and no mutation.

- [ ] **Step 2: Run a small resumable migration without deletion**

```bash
.venv/bin/python scripts/migrate_word_clean_tts_to_minio.py --limit 20 --workers 4 --batch-size 10
```

Expected: 20 uploaded/reused, 20 database rows switched to MinIO, zero deletion.

- [ ] **Step 3: Re-run the small command to prove resume behavior**

Run the same command again.

Expected: no corruption or duplicate records; the command selects the next legacy rows or reports no work according to the documented limit semantics.

- [ ] **Step 4: Run the full upload and database migration without deletion**

```bash
.venv/bin/python scripts/migrate_word_clean_tts_to_minio.py --workers 12 --batch-size 250
```

Expected: all 21,888 success rows use bucket `ai-file-navigation`, keys under `word_clean_tts/`, relative MinIO proxy URLs, and verified objects.

- [ ] **Step 5: Run full verification and guarded deletion**

```bash
.venv/bin/python scripts/migrate_word_clean_tts_to_minio.py --delete-local --sample-size 20 --verify-proxy-base-url http://127.0.0.1:7001
```

Expected: full verification passes before deletion, exactly 21,888 DB-referenced local WAV files are unlinked, and the source directory remains.

- [ ] **Step 6: Independently audit postconditions**

Query PostgreSQL for success/pending counts and legacy URLs; list MinIO keys and compare sizes; count local `.wav` files; fetch at least five evenly distributed proxy URLs and validate HTTP status, content type, and RIFF/WAVE bytes.

Expected: success 21,888, pending 210, legacy success URLs 0, missing or mismatched MinIO objects 0, DB-referenced local files 0, and all five proxy samples pass.

### Task 5: Final verification record

**Files:**
- No repository files expected unless a defect is found.

- [ ] **Step 1: Capture final evidence**

Record the exact migrated, reused, verified, deleted, pending, failed, proxy-sampled, and local-remaining counts in the task handoff. If any counter is non-zero unexpectedly, stop and do not claim completion.

- [ ] **Step 2: Confirm unrelated worktree changes were untouched**

Run: `git status --short`

Expected: only the user's pre-existing unrelated changes remain; migration implementation files are committed.
