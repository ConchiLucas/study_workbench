# Word Agent TTS Model Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` to implement this plan task by task. Use `test-driven-development` for every production change and `verification-before-completion` before reporting success.

**Goal:** Add a standalone Xiaomi MiMo TTS configuration page and persistence path for this repository, make every in-scope TTS generation entry read the active database configuration, and preserve the existing five text-model configurations unchanged.

**Architecture:** React manages TTS providers through new Go endpoints backed by `select_english_word.tts_provider_configs`. Word Agent resolves the one enabled, active `mimo-tts` row immediately before each generation, while timeout, SSL, temporary-file, and MinIO settings remain environment-based. The root demonstration script becomes a Word Agent HTTP client, so `MiMoTTSService` is the only code in this repository that calls Xiaomi directly.

**Tech Stack:** Go, Gin, GORM/PostgreSQL, React 18, TypeScript, Ant Design, Python 3.11, FastAPI, Pydantic, psycopg, pytest, Docker Compose.

## Scope Guardrails

- Do not modify `/Users/conchi/workforce/python_workforce/ai-task-center`.
- Do not insert Xiaomi TTS into `ai_provider_configs` or change `/api/ai/config` behavior.
- Keep all five text providers intact: `aliyun-deepseek`, `glm-5`, `kimi-k2-5`, `minimax-m2-5`, and `qwen3-6-flash`.
- Do not modify or delete existing sentence data, audio objects, MinIO migration scripts, or TTS task records.
- Do not add request-level model or voice overrides; the active TTS configuration is authoritative.
- Never print, return, snapshot, or commit the real Xiaomi API key.

## File Map

### Go management API

- Create `word_select_dashboard/server/model/system/sys_tts_config.go`: GORM row and API-safe payload types.
- Create `word_select_dashboard/server/service/system/sys_tts_config.go`: validation, key preservation, transactions, and safe response mapping.
- Create `word_select_dashboard/server/service/system/sys_tts_config_test.go`: normalization, persistence, and secret-handling tests.
- Create `word_select_dashboard/server/api/v1/system/sys_tts_config.go`: GET/POST handlers.
- Create `word_select_dashboard/server/router/system/sys_tts_config.go`: `/api/tts/config` routes.
- Modify the service/API/router `enter.go` files and `initialize/router.go`: wire the new route.
- Modify `initialize/gorm.go` and `initialize/ensure_tables.go`: include the new table in startup migration checks.

### React administration UI

- Create `word_select_dashboard/web-react/src/types/ttsConfig.ts`: TTS provider and payload types.
- Create `word_select_dashboard/web-react/src/lib/ttsConfigApi.ts`: GET/POST client.
- Create `word_select_dashboard/web-react/src/features/ttsConfig.ts`: defaults and validation helpers.
- Create `word_select_dashboard/web-react/test/ttsConfig.test.ts`: helper and API contract tests.
- Create `word_select_dashboard/web-react/test/ttsConfigPage.contract.test.ts`: menu/page separation contract.
- Modify `word_select_dashboard/web-react/src/App.tsx`: add a separate `tts-config` page after `ai-config`.
- Modify `word_select_dashboard/web-react/src/styles/app.css` only if existing configuration styles are insufficient.

### Python Word Agent runtime

- Create `word_select_dashboard/word-agent/src/word_agent/services/tts_config.py`: PostgreSQL loader and `TTSConfigError`.
- Create `word_select_dashboard/word-agent/tests/test_tts_config.py`: unique active-row and validation tests.
- Modify `services/mimo_tts.py`: fetch database configuration per generation.
- Modify `core/config.py`: remove the four MiMo provider settings while retaining runtime TTS settings.
- Modify `domain/schemas.py`: remove request-level `model` and `voice` overrides.
- Modify `api/routes.py`: map configuration failures to 500 and upstream failures to 502.
- Modify `tests/test_api.py` and create or update `tests/test_mimo_tts.py`.

### Migration, script, and deployment configuration

- Create `word_select_dashboard/word-agent/scripts/migrate_mimo_tts_config.py`: one-time idempotent secret migration.
- Create `word_select_dashboard/word-agent/tests/test_migrate_mimo_tts_config.py`.
- Modify `scripts/mimo_tts_word.py`: call Word Agent rather than Xiaomi.
- Modify `scripts/README_MIMO_TTS.md` and create `tests/test_mimo_tts_word_script.py`.
- Modify `word_select_dashboard/word-agent/.env.example` and its local `docker-compose.yml`: remove runtime MiMo provider variables after migration.
- Do not modify the separate `ai-task-center` project.

## Task 1: Add the Go TTS model and normalization rules

**Files:**

- Create: `word_select_dashboard/server/model/system/sys_tts_config.go`
- Create: `word_select_dashboard/server/service/system/sys_tts_config.go`
- Test: `word_select_dashboard/server/service/system/sys_tts_config_test.go`

### Step 1: Write failing service tests

Cover duplicate IDs, unsupported types, disabled active provider, exactly one active provider, blank key preservation, and rejection of a new provider without a key:

```go
func TestNormalizeTTSConfigPreservesExistingKey(t *testing.T) {
	input := system.TTSConfigPayload{
		Active: "xiaomi-mimo-tts",
		Providers: []system.TTSProviderPayload{{
			ProviderID: "xiaomi-mimo-tts",
			Label: "Xiaomi MiMo TTS",
			Type: "mimo-tts",
			BaseURL: "https://api.xiaomimimo.com/v1",
			Model: "mimo-v2.5-tts",
			Voice: "Chloe",
			Enabled: true,
		}},
	}

	rows, err := normalizeTTSConfig(input, map[string]string{"xiaomi-mimo-tts": "stored-secret"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "stored-secret", rows[0].ApiKey)
	assert.True(t, rows[0].Active)
}

func TestNormalizeTTSConfigRejectsNewProviderWithoutKey(t *testing.T) {
	_, err := normalizeTTSConfig(validTTSConfigPayload(), nil)
	require.ErrorContains(t, err, "API Key")
}
```

### Step 2: Run the focused test and confirm it fails

```bash
cd word_select_dashboard/server
go test ./service/system -run 'TestNormalizeTTSConfig|TestSafeTTSConfig'
```

Expected: compilation fails because the TTS types and helpers do not exist.

### Step 3: Add the GORM row and safe payload types

Use a separate table and JSON names matching the React contract:

```go
type TTSProviderConfig struct {
	ID         uint      `json:"ID" gorm:"primarykey"`
	ProviderID string    `json:"providerId" gorm:"uniqueIndex;size:120;not null"`
	Label      string    `json:"label" gorm:"size:160;not null"`
	Type       string    `json:"type" gorm:"size:80;not null"`
	BaseURL    string    `json:"baseUrl" gorm:"column:base_url;size:500;not null"`
	ApiKey     string    `json:"-" gorm:"column:api_key;type:text;not null"`
	Model      string    `json:"model" gorm:"size:160;not null"`
	Voice      string    `json:"voice" gorm:"size:120;not null"`
	Enabled    bool      `json:"enabled" gorm:"not null;default:true"`
	Active     bool      `json:"active" gorm:"index;not null;default:false"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (TTSProviderConfig) TableName() string { return "tts_provider_configs" }

type TTSProviderPayload struct {
	ProviderID       string `json:"id"`
	Label            string `json:"label"`
	Type             string `json:"type"`
	BaseURL          string `json:"base_url"`
	ApiKey           string `json:"api_key"`
	ApiKeyConfigured bool   `json:"api_key_configured"`
	Model            string `json:"model"`
	Voice            string `json:"voice"`
	Enabled          bool   `json:"enabled"`
}

type TTSConfigPayload struct {
	Active    string               `json:"active"`
	Providers []TTSProviderPayload `json:"providers"`
}
```

### Step 4: Implement normalization and safe response mapping

The helper trims fields, accepts only `mimo-tts`, requires an HTTP(S) Base URL, rejects duplicates, preserves an existing key for a blank submitted key, and derives row `Active` values only from the top-level ID. Reject a disabled active provider. `buildSafeTTSConfig()` must always return `api_key: ""` and set only the boolean `api_key_configured`.

### Step 5: Run focused tests and commit

```bash
cd word_select_dashboard/server
gofmt -w model/system/sys_tts_config.go service/system/sys_tts_config.go service/system/sys_tts_config_test.go
go test ./service/system
cd ../..
git add word_select_dashboard/server/model/system/sys_tts_config.go word_select_dashboard/server/service/system/sys_tts_config.go word_select_dashboard/server/service/system/sys_tts_config_test.go
git commit -m "feat: add TTS configuration domain model"
```

Expected: all `service/system` tests pass.

## Task 2: Persist TTS configuration and expose safe Go endpoints

**Files:**

- Modify: `word_select_dashboard/server/service/system/sys_tts_config.go`
- Create: `word_select_dashboard/server/api/v1/system/sys_tts_config.go`
- Create: `word_select_dashboard/server/router/system/sys_tts_config.go`
- Modify: `word_select_dashboard/server/service/system/enter.go`
- Modify: `word_select_dashboard/server/api/v1/system/enter.go`
- Modify: `word_select_dashboard/server/router/system/enter.go`
- Modify: `word_select_dashboard/server/initialize/router.go`
- Modify: `word_select_dashboard/server/initialize/gorm.go`
- Modify: `word_select_dashboard/server/initialize/ensure_tables.go`
- Test: `word_select_dashboard/server/service/system/sys_tts_config_test.go`

### Step 1: Add failing persistence tests

Use the repository's existing GORM test helper. If none exists, use an isolated test database driver scoped to tests. Cover full-list replacement, blank-key preservation, rollback on failed insert, and safe GET output:

```go
saved, err := service.SaveConfig(db, input)
require.NoError(t, err)
assert.Equal(t, "xiaomi-mimo-tts", saved.Active)
assert.Empty(t, saved.Providers[0].ApiKey)
assert.True(t, saved.Providers[0].ApiKeyConfigured)

var persisted system.TTSProviderConfig
require.NoError(t, db.First(&persisted, "provider_id = ?", "xiaomi-mimo-tts").Error)
assert.Equal(t, "stored-secret", persisted.ApiKey)
```

Run and confirm the methods are missing:

```bash
cd word_select_dashboard/server
go test ./service/system -run 'TestTTSConfigService'
```

### Step 2: Implement transactional reads and writes

Expose:

```go
type TTSConfigService struct{}

func (TTSConfigService) GetConfig(db *gorm.DB) (system.TTSConfigPayload, error)
func (TTSConfigService) SaveConfig(db *gorm.DB, input system.TTSConfigPayload) (system.TTSConfigPayload, error)
```

Inside one transaction, `SaveConfig` reads existing keys, normalizes the payload, deletes only rows from `tts_provider_configs`, inserts normalized rows, and reads them back. Never interpolate a user value or secret into SQL or errors.

### Step 3: Add GET/POST handlers and route wiring

Match the existing AI response envelope:

```go
func (api *TTSConfigApi) GetConfig(c *gin.Context) {
	config, err := ttsConfigService.GetConfig(global.GVA_DB)
	if err != nil {
		response.FailWithMessage("读取 TTS 配置失败", c)
		return
	}
	response.OkWithData(config, c)
}

func (api *TTSConfigApi) SaveConfig(c *gin.Context) {
	var input system.TTSConfigPayload
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FailWithMessage("TTS 配置格式错误", c)
		return
	}
	config, err := ttsConfigService.SaveConfig(global.GVA_DB, input)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(config, c)
}
```

Register `GET /tts/config` and `POST /tts/config` on the same public group used by `/ai/config`. Add `TTSConfigService`, `TTSConfigApi`, and `TTSConfigRouter` to their `enter.go` group structs.

### Step 4: Add the table to both startup migration paths

Add `system.TTSProviderConfig{}` beside `system.AIProviderConfig{}` in `RegisterTables()` and `ensureTables()`. Do not alter the AI model or AI migration entry.

### Step 5: Format, test, and commit

```bash
cd word_select_dashboard/server
gofmt -w model/system/sys_tts_config.go service/system/sys_tts_config.go service/system/sys_tts_config_test.go api/v1/system/sys_tts_config.go router/system/sys_tts_config.go service/system/enter.go api/v1/system/enter.go router/system/enter.go initialize/router.go initialize/gorm.go initialize/ensure_tables.go
go test ./...
cd ../..
git add word_select_dashboard/server
git commit -m "feat: expose standalone TTS configuration API"
```

Expected: all Go packages pass and the server compiles with the new route.

## Task 3: Add typed React helpers and API client

**Files:**

- Create: `word_select_dashboard/web-react/src/types/ttsConfig.ts`
- Create: `word_select_dashboard/web-react/src/lib/ttsConfigApi.ts`
- Create: `word_select_dashboard/web-react/src/features/ttsConfig.ts`
- Create: `word_select_dashboard/web-react/test/ttsConfig.test.ts`

### Step 1: Write failing tests

Test a valid Xiaomi default, duplicate IDs, disabled active item, missing key on a new item, and exact API paths/methods. Stub `fetch` and prove a blank key stays blank in the POST body so Go can preserve the stored key:

```ts
test("saveTTSConfig posts to the dedicated endpoint", async () => {
  const calls: Array<[RequestInfo | URL, RequestInit | undefined]> = [];
  globalThis.fetch = async (input, init) => {
    calls.push([input, init]);
    return new Response(JSON.stringify({ code: 0, data: validConfig, msg: "" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  await saveTTSConfig(validConfig);
  assert.equal(calls[0][0], "/api/tts/config");
  assert.equal(calls[0][1]?.method, "POST");
  assert.equal(JSON.parse(String(calls[0][1]?.body)).providers[0].api_key, "");
});
```

Run and confirm module-not-found failures:

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/ttsConfig.test.ts
```

### Step 2: Add types, defaults, and validation

```ts
export interface TTSProviderConfig {
  id: string;
  label: string;
  type: "mimo-tts";
  base_url: string;
  api_key: string;
  api_key_configured: boolean;
  model: string;
  voice: string;
  enabled: boolean;
}

export interface TTSConfig {
  active: string;
  providers: TTSProviderConfig[];
}
```

`createDefaultTTSProvider()` returns non-secret Xiaomi defaults and a unique generated ID. `validateTTSConfig()` returns a Chinese error for empty lists, duplicate/blank IDs, missing fields, non-HTTP(S) URLs, missing active row, disabled active row, and a new row with neither `api_key` nor `api_key_configured`.

### Step 3: Add the dedicated API client

Mirror `aiConfigApi.ts`, but use `/tts/config` and TTS types. Keep it separate from the text-model client.

### Step 4: Test, build, and commit

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/ttsConfig.test.ts
npm run build
cd ../..
git add word_select_dashboard/web-react/src/types/ttsConfig.ts word_select_dashboard/web-react/src/lib/ttsConfigApi.ts word_select_dashboard/web-react/src/features/ttsConfig.ts word_select_dashboard/web-react/test/ttsConfig.test.ts
git commit -m "feat: add TTS configuration client contracts"
```

Expected: helper tests pass and the application builds.

## Task 4: Add the separate React TTS configuration page

**Files:**

- Modify: `word_select_dashboard/web-react/src/App.tsx`
- Modify only if needed: `word_select_dashboard/web-react/src/styles/app.css`
- Create: `word_select_dashboard/web-react/test/ttsConfigPage.contract.test.ts`

### Step 1: Write a failing page contract test

Following existing source-contract tests, assert that `PageKey` includes `tts-config` separately from `ai-config`, the sidebar contains both menu items in order, `TTSConfigPage` uses the dedicated API, and all TTS fields/actions exist:

```ts
assert.match(appSource, /type PageKey[\s\S]*"ai-config"[\s\S]*"tts-config"/);
assert.match(appSource, /key: "tts-config"/);
assert.match(appSource, /TTS 模型配置/);
assert.match(appSource, /getTTSConfig\(\)/);
assert.match(appSource, /saveTTSConfig\(/);
```

Run and confirm assertions fail:

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/ttsConfigPage.contract.test.ts
```

### Step 2: Add the page and adjacent menu item

Add `tts-config` to `PageKey`, use a sound-oriented Ant icon, and place the item immediately after the existing model page. Implement independent load, select, edit, default, delete, refresh, and save state.

Behavioral rules:

- Empty API Key means preserve the stored key.
- Show `已配置` from the boolean without placing a secret in the DOM.
- Setting a provider as default also enables it.
- Prevent deleting/disabling the active provider until another enabled provider is selected.
- Validate before POST.
- Do not reuse or mutate AI page state.

### Step 3: Test, build, and commit

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/ttsConfig.test.ts test/ttsConfigPage.contract.test.ts
npm run build
cd ../..
git add word_select_dashboard/web-react/src/App.tsx word_select_dashboard/web-react/test/ttsConfigPage.contract.test.ts
git add word_select_dashboard/web-react/src/styles/app.css
git commit -m "feat: add separate TTS model configuration page"
```

If `app.css` was not changed, omit its `git add`. Expected: tests pass and Vite builds without TypeScript errors.

## Task 5: Load the active TTS provider from PostgreSQL in Word Agent

**Files:**

- Create: `word_select_dashboard/word-agent/src/word_agent/services/tts_config.py`
- Create: `word_select_dashboard/word-agent/tests/test_tts_config.py`

### Step 1: Write failing loader tests

Inject a connection factory so tests never use the developer database. Cover one valid row, no row, two rows, disabled rows being excluded, blank fields, and query failures. Assert error text never contains the key:

```python
def test_load_active_mimo_tts_config_returns_validated_row() -> None:
    connection = FakeConnection(rows=[{
        "provider_id": "xiaomi-mimo-tts",
        "type": "mimo-tts",
        "base_url": "https://api.xiaomimimo.com/v1",
        "api_key": "stored-secret",
        "model": "mimo-v2.5-tts",
        "voice": "Chloe",
    }])
    loader = TTSConfigLoader(settings=fake_settings(), connect=lambda _: connection)

    config = loader.load_active_mimo_config()

    assert config.model == "mimo-v2.5-tts"
    assert config.voice == "Chloe"
    assert connection.execute_count == 1
```

Run and confirm import failure:

```bash
cd word_select_dashboard/word-agent
pytest -q tests/test_tts_config.py
```

### Step 2: Implement the loader

Define an immutable runtime value and explicit exception:

```python
@dataclass(frozen=True, slots=True)
class ActiveTTSConfig:
    provider_id: str
    base_url: str
    api_key: str
    model: str
    voice: str

class TTSConfigError(RuntimeError):
    pass
```

Query at most two rows:

```sql
SELECT provider_id, type, base_url, api_key, model, voice
FROM tts_provider_configs
WHERE type = 'mimo-tts' AND enabled = TRUE AND active = TRUE
ORDER BY id
LIMIT 2
```

Use `settings.select_db_dsn` and the existing DSN normalization helper if one exists. Validate exactly one row and nonblank runtime fields. Strip the trailing slash from `base_url`. Raise only non-secret `TTSConfigError` messages.

### Step 3: Test, lint, and commit

```bash
cd word_select_dashboard/word-agent
pytest -q tests/test_tts_config.py
ruff check src/word_agent/services/tts_config.py tests/test_tts_config.py
cd ../..
git add word_select_dashboard/word-agent/src/word_agent/services/tts_config.py word_select_dashboard/word-agent/tests/test_tts_config.py
git commit -m "feat: load active TTS provider from database"
```

Expected: loader tests and Ruff pass.

## Task 6: Make both Word Agent generation paths use database configuration

**Files:**

- Modify: `word_select_dashboard/word-agent/src/word_agent/services/mimo_tts.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/core/config.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/domain/schemas.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/api/routes.py`
- Modify: `word_select_dashboard/word-agent/tests/test_api.py`
- Create or modify: `word_select_dashboard/word-agent/tests/test_mimo_tts.py`

### Step 1: Write failing service and route tests

Use a fake loader and mocked HTTP transport. Prove:

- The loader runs on every `generate()` call, so a saved configuration applies without restart.
- URL, bearer key, model, and voice all come from the loader.
- Environment MiMo values cannot override or rescue a missing database row.
- `/v1/tts/generate` and post-sentence automatic TTS use the same service.
- `TTSConfigError` maps to 500 and Xiaomi failures map to 502.
- Responses report actual database model and voice.
- Legacy request JSON fields `model` and `voice` cannot override database values.

```python
def test_generate_reloads_database_configuration_for_each_call(tmp_path: Path) -> None:
    loader = SequencedLoader([first_config, second_config])
    service = MiMoTTSService(
        settings=fake_settings(tmp_path),
        config_loader=loader,
        http_client=fake_http,
    )

    first = service.generate(text="hello", output_format="wav")
    second = service.generate(text="world", output_format="wav")

    assert first.model == first_config.model
    assert second.model == second_config.model
    assert loader.calls == 2
```

Run and confirm the new assertions fail:

```bash
cd word_select_dashboard/word-agent
pytest -q tests/test_mimo_tts.py tests/test_api.py
```

### Step 2: Inject and use `TTSConfigLoader`

Construct the loader once, but call it inside every `generate()`. Build the Xiaomi request only from its result:

```python
config = self._config_loader.load_active_mimo_config()
request_url = f"{config.base_url}/chat/completions"
headers = {
    "Authorization": f"Bearer {config.api_key}",
    "Content-Type": "application/json",
}
payload = {
    "model": config.model,
    "messages": self._build_messages(text=text, voice=config.voice, style=style),
}
```

Continue using environment settings for timeout, SSL verification, output directory, database, and MinIO.

### Step 3: Remove provider settings and request overrides

Remove `mimo_api_key`, `mimo_tts_base_url`, `mimo_tts_default_model`, and `mimo_tts_default_voice` from `Settings`. Remove `model` and `voice` from `TTSGenerationRequest`, but keep them in response/result schemas for auditing. Unknown legacy request fields may be ignored for compatibility, but tests must prove they have no effect.

### Step 4: Split configuration and upstream errors

Catch `TTSConfigError` separately in both routes and return 500 with a non-secret configuration message. Keep Xiaomi request/response failures at 502. Preserve the current all-or-nothing sentence generation behavior.

### Step 5: Run full tests, lint, and commit

```bash
cd word_select_dashboard/word-agent
pytest -q
ruff check src tests scripts
cd ../..
git add word_select_dashboard/word-agent/src/word_agent/services/mimo_tts.py word_select_dashboard/word-agent/src/word_agent/core/config.py word_select_dashboard/word-agent/src/word_agent/domain/schemas.py word_select_dashboard/word-agent/src/word_agent/api/routes.py word_select_dashboard/word-agent/tests/test_api.py word_select_dashboard/word-agent/tests/test_mimo_tts.py
git commit -m "feat: route Word Agent TTS through database config"
```

Expected: the full suite passes, including the existing deletion tests for `generate_word_clean_sentences()` and `generate_sentence_guidance()`.

## Task 7: Convert the root MiMo script into a Word Agent client

**Files:**

- Modify: `scripts/mimo_tts_word.py`
- Modify: `scripts/README_MIMO_TTS.md`
- Create: `word_select_dashboard/word-agent/tests/test_mimo_tts_word_script.py`

### Step 1: Write a failing script test

Load the root script using `importlib.util.spec_from_file_location`, mock `urllib.request.urlopen`, and assert:

- Default POST target is `http://127.0.0.1:6017/v1/tts/generate`.
- Payload includes text/style/filename/format, but no API key, model, or voice.
- Returned `downloadUrl` is fetched from Word Agent and written to the output path.
- `--word-agent-url` overrides only the Word Agent origin.
- No `MIMO_API_KEY` lookup or Xiaomi hostname remains in the source.

Run and confirm failure:

```bash
cd word_select_dashboard/word-agent
pytest -q tests/test_mimo_tts_word_script.py
```

### Step 2: Implement the client and update documentation

The new command is:

```text
python scripts/mimo_tts_word.py WORD OUTPUT.wav --word-agent-url http://127.0.0.1:6017 --style natural
```

POST `/v1/tts/generate`, validate JSON, resolve its relative `downloadUrl` against the Word Agent origin, fetch WAV bytes, and write the requested output. Remove direct Xiaomi payload construction, Base64 parsing, API-key lookup, `--model`, and `--voice`.

### Step 3: Test, lint, and commit

```bash
cd word_select_dashboard/word-agent
pytest -q tests/test_mimo_tts_word_script.py
ruff check ../../scripts/mimo_tts_word.py tests/test_mimo_tts_word_script.py
cd ../..
git add scripts/mimo_tts_word.py scripts/README_MIMO_TTS.md word_select_dashboard/word-agent/tests/test_mimo_tts_word_script.py
git commit -m "refactor: generate example TTS through Word Agent"
```

Expected: documentation says Word Agent must be running and TTS must be configured in the dashboard.

## Task 8: Add a safe one-time migration and remove legacy runtime variables

**Files:**

- Create: `word_select_dashboard/word-agent/scripts/migrate_mimo_tts_config.py`
- Create: `word_select_dashboard/word-agent/tests/test_migrate_mimo_tts_config.py`
- Modify: `word_select_dashboard/word-agent/.env.example`
- Modify: `word_select_dashboard/word-agent/docker-compose.yml`

### Step 1: Write failing migration tests

Test with injected connection and environment mappings:

- Missing legacy key fails before a write.
- Valid values upsert `xiaomi-mimo-tts` as enabled and active.
- Re-running updates one row without duplicates.
- `ai_provider_configs` is never queried or modified.
- Verification failure rolls back.
- Captured output contains neither the key nor an Authorization header.

```bash
cd word_select_dashboard/word-agent
pytest -q tests/test_migrate_mimo_tts_config.py
```

Expected: import fails because the migration script does not exist.

### Step 2: Implement the idempotent migration

Read current legacy aliases only in this one-time script. Use a parameterized PostgreSQL upsert into `tts_provider_configs`. In the same transaction, mark other TTS rows inactive, read back only non-secret fields plus `api_key <> '' AS api_key_configured`, validate, and commit. Print only provider ID, model, voice, and configured status.

The core upsert is:

```sql
INSERT INTO tts_provider_configs
    (provider_id, label, type, base_url, api_key, model, voice, enabled, active, created_at, updated_at)
VALUES
    (%s, %s, %s, %s, %s, %s, %s, TRUE, TRUE, NOW(), NOW())
ON CONFLICT (provider_id) DO UPDATE SET
    label = EXCLUDED.label,
    type = EXCLUDED.type,
    base_url = EXCLUDED.base_url,
    api_key = EXCLUDED.api_key,
    model = EXCLUDED.model,
    voice = EXCLUDED.voice,
    enabled = TRUE,
    active = TRUE,
    updated_at = NOW()
```

### Step 3: Remove normal runtime provider variables

Remove MiMo key/Base URL/model/voice examples and direct Compose mapping. Retain timeout, SSL, output directory, database DSN, and MinIO settings. Do not edit untracked local secret files; after migration, old values may remain locally but runtime code ignores them.

### Step 4: Test, scan, and commit

```bash
cd word_select_dashboard/word-agent
pytest -q tests/test_migrate_mimo_tts_config.py
ruff check scripts/migrate_mimo_tts_config.py tests/test_migrate_mimo_tts_config.py
cd ../..
rg -n 'MIMO_API_KEY|WORD_AGENT_MIMO_API_KEY|xiaomimimo\.com' word_select_dashboard/word-agent scripts
git add word_select_dashboard/word-agent/scripts/migrate_mimo_tts_config.py word_select_dashboard/word-agent/tests/test_migrate_mimo_tts_config.py word_select_dashboard/word-agent/.env.example word_select_dashboard/word-agent/docker-compose.yml
git commit -m "feat: migrate MiMo TTS credentials into database"
```

Expected: tests and Ruff pass. Remaining Xiaomi hostname matches are non-secret defaults in the migration path or documentation; normal runtime code reads no MiMo key variable.

## Task 9: Run full regression verification before Docker deployment

**Files:** Verify only. Fix failures in the smallest owning file and add a regression test before changing production behavior.

### Step 1: Verify Go

```bash
cd word_select_dashboard/server
go test ./...
```

Expected: all Go packages pass.

### Step 2: Verify React

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/*.test.ts
npm run build
```

Expected: all Node tests pass and the production build succeeds.

### Step 3: Verify Word Agent

```bash
cd word_select_dashboard/word-agent
pytest -q
ruff check src tests scripts ../../scripts/mimo_tts_word.py
```

Expected: full pytest and Ruff suites pass.

### Step 4: Verify scope and secret safety

```bash
cd /Users/conchi/workforce/rob_english_word_workforce
git diff --name-only -- /Users/conchi/workforce/python_workforce/ai-task-center
git diff -- word_select_dashboard/server/model/system/sys_ai_config.go word_select_dashboard/server/service/system/sys_ai_config.go
rg -n 'api[_-]?key|Authorization|MIMO_API_KEY' docs word_select_dashboard scripts
```

Expected: no `ai-task-center` changes and no unintended text-model configuration diff. Manually inspect secret-scan matches; committed fixtures may use explicit dummy strings, but no real key may appear.

### Step 5: Commit only necessary verification fixes

```bash
git status --short
```

If verification required a fix, stage only this feature's files and commit with a message describing the repaired contract. Do not stage unrelated pre-existing worktree changes.

## Task 10: Migrate and deploy in dependency order

**Files:** Operational commands only. Never expose the key in command text or output.

### Step 1: Record the five text providers before migration

Using the documented database connection, query only IDs:

```sql
SELECT provider_id
FROM ai_provider_configs
ORDER BY provider_id;
```

Expected: exactly the five IDs in Scope Guardrails.

### Step 2: Deploy Go first so AutoMigrate creates the table

```bash
sh deploy/backend/word_select_dashboard/local_full/start.sh
```

Expected: the Go container becomes healthy and `tts_provider_configs` exists.

### Step 3: Build Word Agent without replacing the running service

```bash
sh deploy/backend/word_agent/build_project/start.sh
```

Expected: the local image builds successfully.

### Step 4: Run the one-time migration in an ephemeral container

```bash
docker compose -p word-agent -f deploy/backend/word_agent/local_full/docker-compose.yml run --rm app python scripts/migrate_mimo_tts_config.py
```

Expected: output reports `xiaomi-mimo-tts`, `mimo-v2.5-tts`, `Chloe`, and `api_key_configured=true` without printing the key. If migration fails, do not replace the existing Word Agent container.

### Step 5: Verify both configuration tables before switching

```sql
SELECT provider_id, type, model, voice, enabled, active, api_key <> '' AS api_key_configured
FROM tts_provider_configs
ORDER BY provider_id;

SELECT provider_id
FROM ai_provider_configs
ORDER BY provider_id;
```

Expected: one enabled/active Xiaomi MiMo row with a configured key and the unchanged five AI provider IDs.

### Step 6: Start Word Agent and dashboard frontend

```bash
sh deploy/backend/word_agent/local_full/start.sh
sh deploy/frontend/word_select_dashboard/local_full/start.sh
```

Expected: Word Agent is healthy on port 6017 and dashboard is available on port 6016.

### Step 7: Verify API key masking

```bash
curl -fsS http://127.0.0.1:6015/api/tts/config
```

Expected: response contains `"api_key":""` and `"api_key_configured":true`; it does not contain the migrated secret.

### Step 8: Verify TTS flows without uncontrolled billing

Run mocked integration tests inside the deployed image first. If an actual Xiaomi call is authorized for this deployment, make exactly one short `/v1/tts/generate` request, verify returned model/voice, download the WAV, and stop. Then use isolated test data to exercise sentence generation and verify the same metadata.

Do not repeat external calls merely to compare audio quality.

### Step 9: Verify the UI manually

Open `http://127.0.0.1:6016` and confirm:

- “模型配置” still shows five text providers.
- “TTS 模型配置” is a separate adjacent item.
- Xiaomi MiMo TTS loads as enabled and default.
- API Key shows configured state without its value.
- A non-secret edit saves and survives refresh; restore any field changed only for verification.

### Step 10: Commit only tracked deployment fixes

If deployment exposed a tracked fix, stage only the affected feature files and commit `fix: harden TTS configuration deployment`. Do not commit audio, `.env`, dumps, Docker artifacts, or unrelated work.

## Task 11: Final review and handoff

### Step 1: Review the complete feature diff

```bash
git diff --stat a3aaba9..HEAD
git diff --check a3aaba9..HEAD
git status --short
```

Expected: no whitespace errors. Separate pre-existing dirty files from feature files.

### Step 2: Confirm design coverage

Check each requirement against evidence:

- Separate table and endpoints.
- Separate React menu/page.
- Existing five text providers unchanged.
- Database lookup for both Word Agent TTS paths.
- Root script goes through Word Agent.
- No request-level model/voice override.
- API key masking and blank-submit preservation.
- Idempotent, secret-safe migration.
- No `ai-task-center` changes.

### Step 3: Scan for incomplete implementation markers

```bash
rg -n 'TODO|FIXME|XXX|NotImplementedError|pass\s*$' word_select_dashboard/server word_select_dashboard/web-react/src word_select_dashboard/word-agent/src word_select_dashboard/word-agent/scripts scripts/mimo_tts_word.py
```

Expected: no new incomplete marker from this feature. Report unrelated existing matches instead of changing them silently.

### Step 4: Report evidence

The final handoff states files/components changed, exact Go/React/Python/Docker results, database row counts without secrets, whether a real Xiaomi call was made and how many, and any remaining operational action.
