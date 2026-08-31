# Russian Best Sentence Regeneration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Regenerate the canonical `Russian` best sentence through the existing word-agent, create Xiaomi MiMo TTS audio in MinIO, and atomically update the existing best-sentence row.

**Architecture:** This is a one-record operational data change with no source-code modification. The word-agent owns LLM sentence generation, Xiaomi MiMo TTS, and MinIO upload; PostgreSQL is updated only after validating the complete response.

**Tech Stack:** FastAPI word-agent, Xiaomi MiMo TTS, MinIO, PostgreSQL 16, curl, psql

---

### Task 1: Preflight the target and generation services

**Files:**
- Reference: `docs/superpowers/specs/2026-07-18-russian-best-sentence-regeneration-design.md`
- Reference: `word_select_dashboard/word-agent/src/word_agent/api/routes.py`

- [ ] **Step 1: Verify word-agent health**

Run:

```bash
curl -sS http://127.0.0.1:8010/health
```

Expected: HTTP 200 JSON containing `"status":"ok"`.

- [ ] **Step 2: Read the current canonical row**

Run a read-only PostgreSQL query for `word_clean_id = 17241` and `word = 'Russian'`.

Expected: exactly one `word_clean_best_sentence` row with nonempty current sentence and TTS URL.

### Task 2: Generate and validate the replacement sentence and audio

**Files:**
- Reference: `word_select_dashboard/word-agent/src/word_agent/domain/schemas.py`

- [ ] **Step 1: Call the combined generation endpoint**

Run:

```bash
curl -sS -X POST http://127.0.0.1:8010/v1/sentences/generate \
  -H 'Content-Type: application/json' \
  -d '{"words":["Russian"]}'
```

Expected: response contains `sentence`, `translationZh`, `sentenceAudioUrl`, `sentenceAudioBucket`, `sentenceAudioObjectKey`, `ttsProvider`, `ttsModel`, `ttsVoice`, and `ttsFormat`.

- [ ] **Step 2: Validate semantics and storage metadata**

Accept only when the sentence contains the standalone token `Russian`, the Chinese translation has the correct “俄罗斯的/俄语” sense, and all MinIO identity fields are nonempty.

Expected: validation passes without changing PostgreSQL.

### Task 3: Atomically update the canonical best sentence

**Files:**
- Modify data: `rob_english_word.public.word_clean_best_sentence`

- [ ] **Step 1: Build the cloze data**

Replace only the standalone `Russian` token in the generated sentence with `____`; set `cloze_answer = 'Russian'`.

- [ ] **Step 2: Run guarded transaction**

Run a PostgreSQL transaction that first asserts exactly one target row with `word_clean_id = 17241` and `word = 'Russian'`, then updates the generated sentence, translation, cloze fields, and returned TTS/MinIO metadata.

Expected: exactly one row updated and transaction committed.

### Task 4: Verify database and MinIO result

**Files:**
- Verify data: `rob_english_word.public.word_clean_best_sentence`

- [ ] **Step 1: Query the updated row**

Expected: one `Russian` row; sentence, translation, cloze sentence, cloze answer, TTS status, bucket, key, and URL all agree with the generated response.

- [ ] **Step 2: Read the generated audio through the stored URL**

Run an HTTP request against the stored MinIO proxy URL.

Expected: HTTP 200 and nonzero response body.

- [ ] **Step 3: Re-scan the database for the old typo**

Search every `public` base-table column case-insensitively for `2ussian`.

Expected: total remaining occurrences equal 0.
