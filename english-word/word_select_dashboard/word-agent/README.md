# word-agent

Python execution agent for the word selection and sentence-generation workflow.

This project is intentionally only a service framework right now. Go calls the Python service, Python emits step events back to Go, and React reads tracking data from Go.

## Run

```bash
uv sync --extra dev
uv run word-agent
```

The service listens on `http://127.0.0.1:8010` by default.

If `uv` is not installed:

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
word-agent
```

## API Contract

Go starts a run:

```http
POST /v1/runs/execute
```

```json
{
  "runId": "run-20260531-001",
  "word": "brisk",
  "meaning": "quick and energetic",
  "callbackUrl": "http://127.0.0.1:8009/api/python-runs/events",
  "metadata": {
    "source": "go-server"
  }
}
```

Python responds immediately with `202 Accepted`, then sends step events to the callback URL.

```json
{
  "runId": "run-20260531-001",
  "stepId": "prompt_plan",
  "status": "running",
  "message": "开始规划提示词",
  "input": {},
  "output": {},
  "error": null,
  "createdAt": "2026-05-31T13:00:00Z"
}
```

Generate one sentence directly from a few words:

```http
POST /v1/sentences/generate
```

```json
{
  "words": ["brisk", "anchor", "harbor"]
}
```

The service reads the active model provider from `../server/config.yaml`, so Python and Go share the same model config.
`WORD_AGENT_LLM_VERIFY_SSL` defaults to `false` because the current Aliyun endpoint
does not validate cleanly against this local Python certificate chain.

```json
{
  "sentence": "A brisk captain used the anchor to harbor the boat safely.",
  "words": ["brisk", "anchor", "harbor"],
  "providerId": "aliyun-deepseek",
  "providerLabel": "Aliyun DeepSeek V3.2",
  "model": "deepseek-v3.2"
}
```

Generate TTS audio for a word or sentence:

```http
POST /v1/tts/generate
```

```json
{
  "text": "apple",
  "format": "wav"
}
```

For every generation, the service reads the one enabled, active `mimo-tts`
row from `tts_provider_configs`. Model and voice cannot be overridden by the
request. Runtime-only settings such as timeout, SSL verification, and
`WORD_AGENT_TTS_OUTPUT_DIR` remain environment based. The response includes
the model and voice actually used, plus a download URL such as
`/v1/tts/files/apple_20260620120000000000.wav`.

To migrate a legacy local MiMo environment into the new table after the Go
server has created it, run once inside the old environment:

```bash
python scripts/migrate_mimo_tts_config.py
```

The migration output reports only non-secret fields and whether a key is
configured.

Score generated sentences for a small set of saved word-clean sentence rows:

```http
POST /v1/word-clean-sentences/score
```

```json
{
  "wordCleanIds": [48],
  "limit": 4,
  "overwrite": false
}
```

The score is saved back to `word_clean_sentence.score` as a 0-100 integer, with
`score_reason`, `score_model_name`, and `scored_at` for auditability. The
endpoint requires `ids` or `wordCleanIds` so it cannot accidentally score the
whole table. The default judge model is controlled by
`WORD_AGENT_WORD_CLEAN_SCORE_DEFAULT_MODEL`, currently `qwen3.6-flash`.

## Current Scope

- HTTP service skeleton
- typed request/response/event models
- background execution hook
- callback client for Go event ingestion
- simple sentence-generation endpoint backed by the Go AI config
- MiMo TTS endpoint for saving word/sentence audio files
- scoring endpoint for generated `word_clean_sentence` rows
- placeholder execution steps for the async run workflow
