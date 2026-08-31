# Three-Word Cloze TTS and MinIO Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate MiMo TTS for every new three-word cloze sentence, store it in MinIO, and persist its playable URL only when the entire generation succeeds.

**Architecture:** The word-agent sentence endpoint will orchestrate LLM generation, MiMo TTS, MinIO upload, and temporary-file cleanup, returning typed audio metadata with the sentence. The Java cloze service will require the returned audio URL before inserting `sentence_cloze_item`, so the existing wrong-word reservation logic can return all three events to `pending` on any failure.

**Tech Stack:** Python 3.12, FastAPI, Pydantic, MinIO Python SDK, pytest, Java 21, Spring RestClient, MyBatis Plus, JUnit 5, Mockito.

---

## File map

- Create `word_select_dashboard/word-agent/src/word_agent/services/minio_storage.py`: focused MinIO upload and post-upload size verification.
- Create `word_select_dashboard/word-agent/tests/test_minio_storage.py`: unit tests for object naming, URL construction, and upload verification.
- Modify `word_select_dashboard/word-agent/src/word_agent/core/config.py`: typed MinIO settings already present in `.env`.
- Modify `word_select_dashboard/word-agent/.env.example`: document MinIO settings and cloze object prefix.
- Modify `word_select_dashboard/word-agent/pyproject.toml` and `uv.lock`: declare the MinIO SDK dependency.
- Modify `word_select_dashboard/word-agent/src/word_agent/domain/schemas.py`: add sentence audio response fields.
- Modify `word_select_dashboard/word-agent/src/word_agent/api/routes.py`: orchestrate TTS, MinIO, cleanup, and API error mapping.
- Modify `word_select_dashboard/word-agent/tests/test_api.py`: route-level success and failure tests.
- Modify `rob_english_word_back/src/main/java/com/robword/dto/SentenceClozeGenerateResponse.java`: expose the persisted audio URL.
- Modify `rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java`: map, validate, persist, and return sentence audio URL.
- Create `rob_english_word_back/src/test/java/com/robword/service/SentenceClozeServiceTest.java`: service tests using a mock HTTP server and mocked mapper.

### Task 1: MinIO storage boundary

**Files:**
- Create: `word_select_dashboard/word-agent/src/word_agent/services/minio_storage.py`
- Create: `word_select_dashboard/word-agent/tests/test_minio_storage.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/core/config.py`
- Modify: `word_select_dashboard/word-agent/.env.example`
- Modify: `word_select_dashboard/word-agent/pyproject.toml`
- Modify: `word_select_dashboard/word-agent/uv.lock`

- [x] **Step 1: Write failing MinIO storage tests**

Create tests with a fake client that records `fput_object` arguments and returns a `stat_object.size`. Assert that `upload_audio(path, content_type="audio/wav")` uses bucket `ai-file-navigation`, an object key beginning `sentence_cloze_tts/`, preserves `.wav`, returns `/{bucket}/{object_key}`, and raises `MinIOStorageError` when the verified size differs.

- [x] **Step 2: Run tests and confirm red**

Run: `cd word_select_dashboard/word-agent && uv run pytest tests/test_minio_storage.py -v`

Expected: FAIL because `word_agent.services.minio_storage` does not exist.

- [x] **Step 3: Add typed settings and storage implementation**

Add settings fields `minio_endpoint`, `minio_access_key_id`, `minio_secret_access_key`, `minio_use_ssl`, `minio_bucket_name`, `minio_base_path`, and `cloze_tts_object_prefix="sentence_cloze_tts"`. Implement:

```python
@dataclass(frozen=True)
class MinIOUploadResult:
    bucket: str
    object_key: str
    object_url: str
    byte_size: int
    content_type: str

class MinIOStorage:
    def upload_audio(self, file_path: Path, *, content_type: str) -> MinIOUploadResult:
        object_key = self._unique_object_key(file_path)
        self.client.fput_object(bucket, object_key, str(file_path), content_type=content_type)
        stat = self.client.stat_object(bucket, object_key)
        if stat.size != file_path.stat().st_size:
            raise MinIOStorageError(
                f"MinIO object size mismatch: local={file_path.stat().st_size}, remote={stat.size}"
            )
        return MinIOUploadResult(
            bucket=bucket,
            object_key=object_key,
            object_url=f"/{bucket}/{object_key}",
            byte_size=stat.size,
            content_type=content_type,
        )
```

Use a UUID suffix for uniqueness, normalize prefix slashes, verify bucket existence, and never log credentials.

- [x] **Step 4: Declare dependency and document environment**

Add `minio>=7.2.0` to `pyproject.toml`, update `uv.lock`, and add the existing MinIO environment variables plus `WORD_AGENT_CLOZE_TTS_OBJECT_PREFIX=sentence_cloze_tts` to `.env.example`.

- [x] **Step 5: Run focused tests and lint**

Run: `cd word_select_dashboard/word-agent && uv run pytest tests/test_minio_storage.py -v && uv run ruff check src/word_agent/services/minio_storage.py tests/test_minio_storage.py src/word_agent/core/config.py`

Expected: all tests pass and Ruff reports no errors.

### Task 2: Sentence endpoint TTS orchestration

**Files:**
- Modify: `word_select_dashboard/word-agent/src/word_agent/domain/schemas.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/api/routes.py`
- Modify: `word_select_dashboard/word-agent/tests/test_api.py`

- [x] **Step 1: Extend route tests first**

Change the existing sentence test to fake `MiMoTTSService.generate` and `MinIOStorage.upload_audio`. Assert the response includes:

```json
{
  "sentenceAudioUrl": "/ai-file-navigation/sentence_cloze_tts/example.wav",
  "sentenceAudioBucket": "ai-file-navigation",
  "sentenceAudioObjectKey": "sentence_cloze_tts/example.wav",
  "sentenceAudioByteSize": 4,
  "sentenceAudioContentType": "audio/wav",
  "ttsProvider": "xiaomi-mimo",
  "ttsModel": "mimo-v2.5-tts",
  "ttsVoice": "Chloe",
  "ttsFormat": "wav"
}
```

Add tests where TTS raises and where MinIO raises. Assert HTTP 502, MinIO is not called after TTS failure, and the temporary WAV is removed after both successful upload and failed upload.

- [x] **Step 2: Run route tests and confirm red**

Run: `cd word_select_dashboard/word-agent && uv run pytest tests/test_api.py::test_generate_sentence_uses_llm_tts_and_minio -v`

Expected: FAIL because response fields and orchestration do not exist.

- [x] **Step 3: Add typed response fields**

Extend `SentenceGenerationResponse` with aliased fields for URL, bucket, key, byte size, content type, TTS provider/model/voice/format. Keep existing sentence fields unchanged for compatibility.

- [x] **Step 4: Implement orchestration with unconditional cleanup**

After LLM generation, build a `TTSGenerationRequest(text=result.sentence)`, generate the WAV, upload it, then return both sentence and audio metadata. Wrap the upload in:

```python
tts_result = await tts_service.generate(tts_request)
try:
    upload = storage.upload_audio(tts_result.file_path, content_type=tts_result.content_type)
finally:
    tts_result.file_path.unlink(missing_ok=True)
```

Map `TTSConfigError`, `TTSRequestError`, and `MinIOStorageError` to HTTP 502 for this composite endpoint. The standalone TTS endpoint remains unchanged.

- [x] **Step 5: Run word-agent suite**

Run: `cd word_select_dashboard/word-agent && uv run pytest -v && uv run ruff check src tests`

Expected: all tests pass and Ruff reports no errors.

### Task 3: Java validation and persistence

**Files:**
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/SentenceClozeGenerateResponse.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/SentenceClozeService.java`
- Create: `rob_english_word_back/src/test/java/com/robword/service/SentenceClozeServiceTest.java`

- [x] **Step 1: Write failing service tests**

Use JDK `HttpServer` as the fake word-agent and Mockito for `SentenceClozeItemMapper`. Set `wordAgentBaseUrl` through `ReflectionTestUtils`. Test a successful JSON response with `sentenceAudioUrl` and capture the inserted entity:

```java
verify(mapper).insert(itemCaptor.capture());
assertEquals("/ai-file-navigation/sentence_cloze_tts/example.wav",
        itemCaptor.getValue().getSentenceAudioUrl());
```

Add a missing-audio response test and assert:

```java
assertThrows(IllegalStateException.class, () -> service.generateAndSave(request));
verify(mapper, never()).insert(any(SentenceClozeItem.class));
```

- [x] **Step 2: Run target test and confirm red**

Run: `cd rob_english_word_back && ./mvnw -Dtest=SentenceClozeServiceTest test`

Expected: FAIL because the service does not map, validate, persist, or return the audio URL.

- [x] **Step 3: Implement response mapping and validation**

Add `sentenceAudioUrl` to the private word-agent response and public DTO. Normalize it using a method that rejects null/blank values:

```java
private String requireSentenceAudioUrl(String value) {
    if (value == null || value.isBlank()) {
        throw new IllegalStateException("word-agent 未返回挖空句子 TTS 音频地址");
    }
    return value.trim();
}
```

Call this before constructing and inserting the entity, then set the value on both the entity and returned DTO.

- [x] **Step 4: Run Java tests**

Run: `cd rob_english_word_back && ./mvnw -Dtest=SentenceClozeServiceTest test`

Expected: all target tests pass.

### Task 4: End-to-end verification and operational check

**Files:**
- Modify: `progress.md`
- Modify: `findings.md`

- [x] **Step 1: Run complete relevant verification**

Run:

```bash
cd word_select_dashboard/word-agent && uv run pytest -v && uv run ruff check src tests
cd rob_english_word_back && ./mvnw test
git diff --check
```

Expected: Python suite, Java suite, Ruff, and whitespace validation all pass.

- [x] **Step 2: Restart services and check health**

Restart word-agent on 8010 and Java backend on 8019 using their project scripts, then verify health endpoints respond. Do not synthesize a real MiMo sentence unless explicitly needed; automated tests use fakes and do not consume paid API quota.

- [x] **Step 3: Record outcomes**

Append exact files changed, test counts, and any operational caveats to `progress.md` and `findings.md`.
