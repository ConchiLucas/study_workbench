# Unified Workspace Runtime Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make root `.env.local` the only machine-specific configuration needed to build, start, and verify all six Docker projects plus the host CLI Runner.

**Architecture:** A standard-library Python resolver validates canonical component fields and derives host/container runtime variables without printing secrets. The root orchestration script loads those variables, all Compose files consume them, Go receives explicit Viper environment bindings, and existing CLI Runner lifecycle helpers remain responsible for the host process.

**Tech Stack:** POSIX shell/zsh, Python 3 standard library and pytest, Docker Compose, Go/Viper, nginx official image templates, Spring Boot environment binding.

---

### Task 1: Canonical root configuration and resolver

**Files:**
- Create: `.env.example`
- Create: `scripts/workspace_runtime_config.py`
- Create: `word_select_dashboard/word-agent/tests/test_workspace_runtime_config.py`
- Modify: `.gitignore`

- [ ] **Step 1: Write failing resolver tests**

Cover required-field aggregation, loopback conversion, remote-host preservation, URI encoding, atomic `0600` output, and secret-free errors.

```python
def test_derive_runtime_uses_one_host_for_host_and_container() -> None:
    values = valid_values(SELECT_DB_HOST="127.0.0.1")
    runtime = resolver.derive_runtime(values)
    assert runtime["SELECT_DB_HOST_RUNTIME"] == "127.0.0.1"
    assert runtime["SELECT_DB_CONTAINER_HOST"] == "host.docker.internal"
    assert "host.docker.internal" in runtime["WORD_AGENT_SELECT_DB_DSN"]
    assert "127.0.0.1" in runtime["WORD_AGENT_CLI_RUNNER_DB_DSN"]


def test_missing_fields_are_reported_together() -> None:
    with pytest.raises(resolver.ConfigError) as exc_info:
        resolver.derive_runtime({})
    assert "SELECT_DB_HOST" in str(exc_info.value)
    assert "REDIS_HOST" in str(exc_info.value)
    assert "MINIO_HOST" in str(exc_info.value)
```

- [ ] **Step 2: Run tests and verify failure**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_workspace_runtime_config.py
```

Expected: FAIL because the resolver does not exist.

- [ ] **Step 3: Implement resolver and root template**

Expose focused functions:

```python
class ConfigError(RuntimeError):
    pass


def container_host(host: str) -> str:
    normalized = host.strip()
    if normalized.lower() in {"127.0.0.1", "localhost", "::1"}:
        return "host.docker.internal"
    return normalized


def derive_runtime(values: Mapping[str, str]) -> dict[str, str]:
    """Validate canonical fields and return host/container runtime variables."""


def write_runtime_environment(path: Path, values: Mapping[str, str]) -> None:
    """Atomically write shell-compatible KEY=value lines with mode 0600."""
```

Use `urllib.parse.quote` for PostgreSQL URIs. Derive host/container PostgreSQL DSNs, Spring JDBC values, Go PostgreSQL/Redis/MinIO variables, Word Agent dependency variables, service ports, and host/container URLs. The CLI accepts `--output`, reads exported canonical values from `os.environ`, writes only the runtime file, and prints no values.

Add all canonical fields to root `.env.example`; add `.runtime-config/` to `.gitignore`.

- [ ] **Step 4: Run tests and lint**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_workspace_runtime_config.py
word_select_dashboard/word-agent/.venv/bin/ruff check \
  scripts/workspace_runtime_config.py \
  word_select_dashboard/word-agent/tests/test_workspace_runtime_config.py
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .env.example .gitignore scripts/workspace_runtime_config.py \
  word_select_dashboard/word-agent/tests/test_workspace_runtime_config.py
git commit -m "feat: add unified workspace runtime config"
```

### Task 2: Make CLI bootstrap consume unified runtime variables

**Files:**
- Modify: `scripts/bootstrap_sentence_cli_configs.py`
- Modify: `word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py`

- [ ] **Step 1: Write failing tests**

```python
def test_resolve_database_dsns_prefers_runtime_environment(monkeypatch, tmp_path):
    monkeypatch.setenv("WORD_AGENT_CLI_RUNNER_DB_DSN", "postgresql://host-db")
    monkeypatch.setenv("WORD_AGENT_SELECT_DB_DSN", "postgresql://container-db")
    assert bootstrap.resolve_database_dsns(tmp_path / "missing.yaml") == (
        "postgresql://host-db",
        "postgresql://container-db",
    )


def test_cli_provider_paths_use_runtime_environment(monkeypatch, tmp_path):
    monkeypatch.setenv("CODEX_COMMAND_PATH", "/opt/bin/codex")
    monkeypatch.setenv("GEMINI_COMMAND_PATH", "/opt/bin/gemini")
    connection = FakeConnection()
    bootstrap.insert_missing_cli_configs(connection, project_root=tmp_path)
    assert connection.cursor_instance.calls[1][1][3] == "/opt/bin/codex"
    assert connection.cursor_instance.calls[2][1][3] == "/opt/bin/gemini"
```

- [ ] **Step 2: Verify focused tests fail**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py \
  -k 'runtime_environment'
```

Expected: FAIL because YAML and fixed command paths still win.

- [ ] **Step 3: Implement environment precedence**

Return both non-empty environment DSNs before reading YAML; reject partial environment DSN configuration. Resolve provider paths using:

```python
codex_path = os.environ.get("CODEX_COMMAND_PATH", CODEX_COMMAND_PATH).strip()
gemini_path = os.environ.get("GEMINI_COMMAND_PATH", GEMINI_COMMAND_PATH).strip()
```

Keep values out of messages.

- [ ] **Step 4: Run the full bootstrap test file**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add scripts/bootstrap_sentence_cli_configs.py \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py
git commit -m "feat: load cli runner config from workspace env"
```

### Task 3: Add explicit Go environment overrides

**Files:**
- Modify: `word_select_dashboard/server/core/viper.go`
- Modify: `word_select_dashboard/server/core/viper_test.go`
- Modify: `word_select_dashboard/server/Dockerfile`

- [ ] **Step 1: Write failing binding tests**

Set runtime variables, bind them to a minimal YAML, unmarshal, and assert environment values win.

```go
func TestBindRuntimeEnvironmentOverridesMachineSpecificConfig(t *testing.T) {
    t.Setenv("SELECT_DB_CONTAINER_HOST", "host.docker.internal")
    t.Setenv("SELECT_DB_PORT", "5544")
    t.Setenv("SELECT_DB_NAME", "portable_select")
    t.Setenv("SELECT_DB_USER", "portable_user")
    t.Setenv("SELECT_DB_PASSWORD", "portable_secret")
    t.Setenv("REDIS_CONTAINER_ADDR", "host.docker.internal:6380")
    t.Setenv("WORD_AGENT_CONTAINER_URL", "http://host.docker.internal:6017")
    // bind, read fixture, unmarshal, and assert overrides.
}
```

- [ ] **Step 2: Verify Go test failure**

```bash
cd word_select_dashboard/server
go test ./core -run RuntimeEnvironment -count=1
```

Expected: FAIL because `bindRuntimeEnvironment` does not exist.

- [ ] **Step 3: Implement bindings and remove Dockerfile rewrites**

```go
func bindRuntimeEnvironment(v *viper.Viper) error {
    bindings := map[string]string{
        "pgsql.path": "SELECT_DB_CONTAINER_HOST",
        "pgsql.port": "SELECT_DB_PORT",
        "pgsql.db-name": "SELECT_DB_NAME",
        "pgsql.username": "SELECT_DB_USER",
        "pgsql.password": "SELECT_DB_PASSWORD",
        "redis.addr": "REDIS_CONTAINER_ADDR",
        "redis.password": "REDIS_PASSWORD",
        "minio.endpoint": "MINIO_CONTAINER_ENDPOINT",
        "minio.access-key-id": "MINIO_ACCESS_KEY",
        "minio.secret-access-key": "MINIO_SECRET_KEY",
        "minio.bucket-name": "MINIO_BUCKET",
        "minio.use-ssl": "MINIO_USE_SSL",
        "word-agent.base-url": "WORD_AGENT_CONTAINER_URL",
        "system.addr": "DASHBOARD_SERVER_PORT",
    }
    for key, environment := range bindings {
        if err := v.BindEnv(key, environment); err != nil {
            return err
        }
    }
    return nil
}
```

Call it before `ReadInConfig`. Delete Dockerfile `awk`/`sed` credential and port rewrites.

- [ ] **Step 4: Run Go tests**

```bash
cd word_select_dashboard/server
go test ./core ./config -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add word_select_dashboard/server/core/viper.go \
  word_select_dashboard/server/core/viper_test.go \
  word_select_dashboard/server/Dockerfile
git commit -m "feat: override dashboard config from workspace env"
```

### Task 4: Make all six Compose projects consume the root runtime

**Files:**
- Modify: `word_select_dashboard/word-agent/docker-compose.yml`
- Modify: `word_select_dashboard/server/docker-compose.yml`
- Modify: `rob_english_word_back/docker-compose.yml`
- Modify: `rob_english_word_front/docker-compose.yml`
- Modify: `rob_english_word_cloze_web/docker-compose.yml`
- Modify: `word_select_dashboard/web-react/docker-compose.yml`
- Modify: all three frontend `Dockerfile` and `nginx.conf`
- Modify: `word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py`

- [ ] **Step 1: Write failing static contract tests**

```python
def test_full_compose_projects_use_root_runtime_contract() -> None:
    agent = COMPOSE_PATH.read_text(encoding="utf-8")
    assert "${WORD_AGENT_SELECT_DB_DSN:?" in agent
    assert "../server/config.docker.yaml:/app/server-config/config.yaml:ro" in agent
    assert "server/config.yaml" not in agent
```

Also assert Java uses `WORD_AGENT_CONTAINER_URL`, Go receives explicit runtime fields, frontend upstreams are templated, and versioned runtime files contain neither fixed credentials nor `host.docker.internal:8010`.

- [ ] **Step 2: Verify tests fail**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py \
  -k 'compose or runtime_contract'
```

Expected: FAIL.

- [ ] **Step 3: Update backend Compose files**

Use required interpolation for secrets and derived values:

```yaml
environment:
  WORD_AGENT_SELECT_DB_DSN: "${WORD_AGENT_SELECT_DB_DSN:?runtime config is required}"
  WORD_AGENT_ROB_WORD_DB_DSN: "${WORD_AGENT_ROB_WORD_DB_DSN:?runtime config is required}"
  WORD_AGENT_CLI_RUNNER_URL: "${WORD_AGENT_CONTAINER_CLI_RUNNER_URL:?runtime config is required}"
```

Add `host.docker.internal:host-gateway` where a container calls host services. Parameterize host ports while preserving current defaults.

- [ ] **Step 4: Update nginx templates and frontend Compose**

Copy each nginx configuration to `/etc/nginx/templates/default.conf.template`; replace upstream host ports with nginx-supported environment placeholders. Pass only required upstream variables in Compose.

- [ ] **Step 5: Run contract tests**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add word_select_dashboard rob_english_word_back rob_english_word_front \
  rob_english_word_cloze_web
git commit -m "feat: connect all compose projects to root config"
```

### Task 5: Replace root full-deploy orchestration

**Files:**
- Modify: `deploy-compose-full.sh`
- Modify: `scripts/start_word_select_dashboard.sh`
- Modify: `word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py`
- Create: `word_select_dashboard/word-agent/tests/test_deploy_compose_full.py`

- [ ] **Step 1: Write failing orchestration tests**

Use fake commands and run from a temporary cwd. Assert root resolution, aggregate preflight, six Compose config checks, no build in `--check`, CLI Runner ordering, complete probes, and secret-free output.

- [ ] **Step 2: Verify tests fail**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_deploy_compose_full.py
```

Expected: FAIL because the current script uses `pwd` and has no check mode.

- [ ] **Step 3: Implement canonical loading and preflight**

```sh
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CONFIG_FILE=${WORKSPACE_ENV_FILE:-"$ROOT_DIR/.env.local"}
RUNTIME_DIR="$ROOT_DIR/.runtime-config"
RUNTIME_ENV="$RUNTIME_DIR/full-compose.env"
```

Load root config with `set -a`, run the resolver, load its `0600` output, then check required tools, ports, dependencies, and all six `docker compose config -q` commands. `--check` exits here.

- [ ] **Step 4: Integrate CLI Runner and six projects**

Adapt `start_word_select_dashboard.sh` to accept derived runtime variables and skip ignored YAML resolution. Preserve launchd/nohup PID safety. Build the Word Agent base image, start dashboard trio/CLI Runner, then Java, Vue, and cloze.

- [ ] **Step 5: Add bounded readiness and failure summary**

Probe six containers, three backend/Agent endpoints, three frontend endpoints, PostgreSQL, Redis, and MinIO. On failure print safe status/log commands; never run `down`, remove containers, or reveal expanded DSNs.

- [ ] **Step 6: Run focused tests**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q \
  word_select_dashboard/word-agent/tests/test_deploy_compose_full.py \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py \
  word_select_dashboard/word-agent/tests/test_workspace_runtime_config.py
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add deploy-compose-full.sh scripts/start_word_select_dashboard.sh \
  word_select_dashboard/word-agent/tests
git commit -m "feat: add portable full compose startup"
```

### Task 6: Documentation, migration, and full Docker verification

**Files:**
- Modify: `docs/shared/runtime-deployment-map.md`
- Runtime-only, ignored: `.env.local`

- [ ] **Step 1: Document the workflow**

```bash
cp .env.example .env.local
$EDITOR .env.local
./deploy-compose-full.sh --check
./deploy-compose-full.sh
```

State that child `.env`/`config.yaml` files are not part of Docker full startup.

- [ ] **Step 2: Create this machine's ignored config safely**

Populate `.env.local` from existing private configuration without printing values; set mode `0600`. Validate key names only.

- [ ] **Step 3: Run static verification**

```bash
./deploy-compose-full.sh --check
git diff --check
git grep -n 'host.docker.internal:8010\|change-me-password' -- \
  '*docker-compose*.yml' '*Dockerfile*'
```

Expected: preflight passes; forbidden runtime defaults are absent.

- [ ] **Step 4: Run complete tests**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest -q
cd word_select_dashboard/server && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Run full Docker build/start**

```bash
./deploy-compose-full.sh
```

Expected: six containers and CLI Runner pass readiness checks. If an external dependency is unavailable, report it without changing configuration.

- [ ] **Step 6: Commit documentation**

```bash
git add docs/shared/runtime-deployment-map.md
git commit -m "docs: document unified workspace startup"
```

Expected: `.env.local` and `.runtime-config/` remain ignored.
