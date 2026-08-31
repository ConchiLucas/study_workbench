# API / CLI Unified Sentence Executor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在同一个“模型配置”页面中管理 API、Codex CLI 和 Gemini CLI，并让 Word Agent 的造句链路始终只使用一个全局生效执行器。

**Architecture:** PostgreSQL 保存 API 配置、CLI 配置和单行造句执行器选择；Go 提供统一事务配置接口，React 用两个 Tab 编辑并显示唯一状态。Docker 中的 Word Agent 负责业务 Prompt 和结果解析；macOS 主机上的独立 CLI Runner 通过固定驱动执行 Codex/Gemini，并返回统一文本。

**Tech Stack:** Go、Gin、GORM、PostgreSQL、React、TypeScript、Ant Design、TanStack Query、Python 3.12、FastAPI、httpx、psycopg、pytest、Docker Compose、zsh。

---

## 文件结构

- Go 模型：`word_select_dashboard/server/model/system/sys_cli_config.go`
- Go 服务：`word_select_dashboard/server/service/system/sys_execution_config.go`
- Go API/路由：`word_select_dashboard/server/api/v1/system/sys_execution_config.go`、`router/system/sys_execution_config.go`
- React 类型/逻辑/API：`src/types/executionConfig.ts`、`src/features/executionConfig.ts`、`src/lib/executionConfigApi.ts`
- React 页面：`src/components/ExecutionConfigPage.tsx`
- Word Agent 执行器：`src/word_agent/services/sentence_executor.py`
- 主机 Runner：`src/word_agent/cli_runner/{config_client,drivers,service,main}.py`
- 项目脚本：`scripts/bootstrap_sentence_cli_configs.py`、`start_word_select_dashboard.sh`、`stop_word_select_dashboard.sh`

## Task 1: 将已部署的 TTS 实现安全带入 `dev`

**Files:**
- Preserve: `word_select_dashboard/word-agent/src/word_agent/services/llm_client.py`
- Preserve: `word_select_dashboard/word-agent/tests/test_llm_client.py`
- Import: commits `0e999cb` through `c65fea8`

- [ ] **Step 1: 确认重叠文件并只 stash 两个文件**

```bash
git diff --name-only bd0cb65..c65fea8
git stash push -m "preserve pre-tts llm edits" -- \
  word_select_dashboard/word-agent/src/word_agent/services/llm_client.py \
  word_select_dashboard/word-agent/tests/test_llm_client.py
```

Expected: 其他用户未提交文件仍留在工作区。

- [ ] **Step 2: 带入 9 个已验证提交**

```bash
git cherry-pick 0e999cb 9bd56bc adcdec0 befa86c d263aad \
  5993a70 34d792a c196563 c65fea8
git stash pop
```

Expected: TTS 实现进入 `dev`；两个 LLM 文件内容未丢失。

- [ ] **Step 3: 运行基线回归**

```bash
(cd word_select_dashboard/server && go test ./...)
(cd word_select_dashboard/web-react && node --test --experimental-strip-types test/*.test.ts && npm run build)
(cd word_select_dashboard/word-agent && .venv/bin/pytest -q)
```

Expected: Go 全部通过、React 13 个基线测试通过、Python 66 个基线测试通过。

## Task 2: 增加 CLI 与唯一造句执行器数据库模型

**Files:**
- Create: `word_select_dashboard/server/model/system/sys_cli_config.go`
- Modify: `word_select_dashboard/server/initialize/gorm.go`
- Modify: `word_select_dashboard/server/initialize/ensure_tables.go`
- Test: `word_select_dashboard/server/initialize/sentence_executor_table_registration_test.go`

- [ ] **Step 1: 写失败的迁移注册测试**

```go
func readInitializationSources(t *testing.T) string {
    t.Helper()
    var source strings.Builder
    for _, name := range []string{"gorm.go", "ensure_tables.go"} {
        content, err := os.ReadFile(name)
        if err != nil { t.Fatal(err) }
        source.Write(content)
    }
    return source.String()
}

func TestSentenceExecutorTablesAreRegistered(t *testing.T) {
    source := readInitializationSources(t)
    for _, model := range []string{"CLIProviderConfig", "SentenceExecutorConfig"} {
        if !strings.Contains(source, "&system."+model+"{}") {
            t.Fatalf("missing migration registration for %s", model)
        }
    }
}
```

- [ ] **Step 2: 验证 RED**

```bash
cd word_select_dashboard/server
go test ./initialize -run TestSentenceExecutorTablesAreRegistered -v
```

Expected: FAIL，提示缺少两个模型。

- [ ] **Step 3: 实现数据库模型和两个迁移入口**

```go
type CLIProviderConfig struct {
    ID               uint      `gorm:"primarykey"`
    ProviderID       string    `gorm:"uniqueIndex;size:120;not null"`
    Label            string    `gorm:"size:160;not null"`
    Driver           string    `gorm:"size:32;not null"`
    CommandPath      string    `gorm:"type:text;not null"`
    Model            string    `gorm:"size:160;not null"`
    ReasoningEffort  string    `gorm:"size:32;not null;default:''"`
    WorkingDirectory string    `gorm:"type:text;not null"`
    TimeoutSeconds   int       `gorm:"not null;default:300"`
    Enabled          bool      `gorm:"not null;default:true"`
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
func (CLIProviderConfig) TableName() string { return "cli_provider_configs" }

type SentenceExecutorConfig struct {
    SingletonKey string    `gorm:"primaryKey;size:32"`
    ExecutorType string    `gorm:"size:16;not null"`
    ExecutorID   string    `gorm:"size:120;not null"`
    UpdatedAt    time.Time
}
func (SentenceExecutorConfig) TableName() string { return "sentence_executor_config" }
```

- [ ] **Step 4: 验证 GREEN 并提交**

```bash
go test ./initialize -run TestSentenceExecutorTablesAreRegistered -v
go test ./...
git add word_select_dashboard/server/model/system/sys_cli_config.go \
  word_select_dashboard/server/initialize
git commit -m "feat: add sentence executor configuration models"
```

Expected: PASS。

## Task 3: 实现统一配置事务服务

**Files:**
- Create: `word_select_dashboard/server/service/system/sys_execution_config.go`
- Create: `word_select_dashboard/server/service/system/sys_execution_config_test.go`
- Modify: `word_select_dashboard/server/service/system/enter.go`

- [ ] **Step 1: 写失败测试：目标必须存在、启用且唯一**

```go
func TestNormalizeExecutionConfigRejectsUnknownTarget(t *testing.T) {
    req := validExecutionConfigInput()
    req.ActiveTarget = ExecutionTarget{Type: "cli", ID: "missing"}
    _, err := NormalizeExecutionConfig(req, nil)
    require.ErrorContains(t, err, "不存在")
}

func TestNormalizeExecutionConfigRejectsDisabledTarget(t *testing.T) {
    req := validExecutionConfigInput()
    req.CLIProviders[0].Enabled = false
    _, err := NormalizeExecutionConfig(req, nil)
    require.ErrorContains(t, err, "已停用")
}
```

- [ ] **Step 2: 验证 RED**

```bash
cd word_select_dashboard/server
go test ./service/system -run ExecutionConfig -v
```

Expected: FAIL with undefined execution config types/functions。

- [ ] **Step 3: 定义输入和固定白名单**

```go
type ExecutionTarget struct { Type string `json:"type"`; ID string `json:"id"` }
type AIProviderInput struct {
    ID string `json:"id"`; Label string `json:"label"`; Type string `json:"type"`
    BaseURL string `json:"base_url"`; APIKey string `json:"api_key"`
    Model string `json:"model"`; MaxTokens int `json:"max_tokens"`
}
type CLIProviderInput struct {
    ID string `json:"id"`; Label string `json:"label"`; Driver string `json:"driver"`
    CommandPath string `json:"command_path"`; Model string `json:"model"`
    ReasoningEffort string `json:"reasoning_effort"`; WorkingDirectory string `json:"working_directory"`
    TimeoutSeconds int `json:"timeout_seconds"`; Enabled bool `json:"enabled"`
}
type ExecutionConfigInput struct {
    ActiveTarget ExecutionTarget `json:"active_target"`
    APIProviders []AIProviderInput `json:"api_providers"`
    CLIProviders []CLIProviderInput `json:"cli_providers"`
}

var cliModelOptions = map[string]map[string]struct{}{
    "codex": {"gpt-5.6-sol": {}, "gpt-5.6-terra": {}, "gpt-5.6-luna": {}, "gpt-5.5": {}, "gpt-5.4": {}, "gpt-5.4-mini": {}, "gpt-5.3-codex-spark": {}},
    "gemini": {"auto": {}, "pro": {}, "flash": {}, "flash-lite": {}},
}
```

Codex 推理强度只允许 `low|medium|high|xhigh`；Gemini 强制为空。路径必须为非空绝对路径，主机可执行性由 Runner 校验。

- [ ] **Step 4: 写失败测试：空 API Key 保留且 CLI 生效时 API active 全清**

```go
func TestSavePreservesBlankKeyAndClearsAPIActiveForCLI(t *testing.T) {
    db := newExecutionConfigTestDB(t)
    seedAIProvider(t, db, "aliyun", "stored-secret", true)
    req := validExecutionConfigInput()
    req.APIProviders[0].ID, req.APIProviders[0].APIKey = "aliyun", ""
    req.ActiveTarget = ExecutionTarget{Type: "cli", ID: "codex"}
    require.NoError(t, new(ExecutionConfigService).Save(db, req))
    assert.Equal(t, "stored-secret", loadAPIKey(t, db, "aliyun"))
    assert.False(t, loadAPIActive(t, db, "aliyun"))
    assert.Equal(t, ExecutionTarget{Type: "cli", ID: "codex"}, loadTarget(t, db))
}
```

- [ ] **Step 5: 实现单事务保存**

```go
return db.Transaction(func(tx *gorm.DB) error {
    if err := saveAPIProviders(tx, normalized.APIProviders, existingKeys); err != nil { return err }
    if err := saveCLIProviders(tx, normalized.CLIProviders); err != nil { return err }
    if err := syncLegacyAPIActive(tx, normalized.ActiveTarget); err != nil { return err }
    row := system.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: normalized.ActiveTarget.Type, ExecutorID: normalized.ActiveTarget.ID}
    return tx.Clauses(clause.OnConflict{
        Columns: []clause.Column{{Name: "singleton_key"}},
        DoUpdates: clause.AssignmentColumns([]string{"executor_type", "executor_id", "updated_at"}),
    }).Create(&row).Error
})
```

`Load` 在 singleton 不存在时只迁移旧的 `active=true` API；没有 active 时返回“尚未选择造句执行器”，禁止按字母顺序回退。

- [ ] **Step 6: 验证 GREEN 并提交**

```bash
go test ./service/system -run ExecutionConfig -v
go test ./...
git add word_select_dashboard/server/service/system/sys_execution_config.go \
  word_select_dashboard/server/service/system/sys_execution_config_test.go \
  word_select_dashboard/server/service/system/enter.go
git commit -m "feat: persist one active sentence executor"
```

Expected: PASS。

## Task 4: 暴露统一 Go 管理 API

**Files:**
- Create: `word_select_dashboard/server/api/v1/system/sys_execution_config.go`
- Test: `word_select_dashboard/server/api/v1/system/sys_execution_config_test.go`
- Create: `word_select_dashboard/server/router/system/sys_execution_config.go`
- Test: `word_select_dashboard/server/router/system/sys_execution_config_test.go`
- Modify: API/router `enter.go` and `word_select_dashboard/server/initialize/router.go`

- [ ] **Step 1: 写失败测试：API Key 脱敏和 GET/POST 路由**

```go
func TestExecutionConfigResponseMasksAPIKeys(t *testing.T) {
    output := buildExecutionConfigResponse(ExecutionConfigInput{
        APIProviders: []AIProviderInput{{ID: "aliyun", APIKey: "secret"}},
    })
    assert.Empty(t, output.APIProviders[0].APIKey)
    assert.True(t, output.APIProviders[0].APIKeyConfigured)
}

func TestExecutionConfigRouterRegistersGETAndPOST(t *testing.T) {
    routes := registeredExecutionConfigRoutes()
    assert.Contains(t, routes, "GET /ai/execution-config")
    assert.Contains(t, routes, "POST /ai/execution-config")
}
```

- [ ] **Step 2: 验证 RED**

```bash
cd word_select_dashboard/server
go test ./api/v1/system ./router/system -run ExecutionConfig -v
```

Expected: FAIL。

- [ ] **Step 3: 实现控制器和路由**

```go
func (a *ExecutionConfigApi) GetConfig(c *gin.Context) {
    config, err := executionConfigService.Load(global.GVA_DB)
    if err != nil { response.FailWithMessage(err.Error(), c); return }
    response.OkWithDetailed(maskExecutionConfig(config), "获取成功", c)
}

func (a *ExecutionConfigApi) SaveConfig(c *gin.Context) {
    var req service.ExecutionConfigInput
    if err := c.ShouldBindJSON(&req); err != nil {
        response.FailWithMessage("参数错误: "+err.Error(), c); return
    }
    saved, err := executionConfigService.SaveAndLoad(global.GVA_DB, req)
    if err != nil { response.FailWithMessage(err.Error(), c); return }
    response.OkWithDetailed(maskExecutionConfig(saved), "保存成功", c)
}
```

路由精确注册 `GET/POST /ai/execution-config`，旧 `/ai/config` 保留兼容。

- [ ] **Step 4: 验证 GREEN 并提交**

```bash
go test ./api/v1/system ./router/system -run ExecutionConfig -v
go test ./...
git add word_select_dashboard/server/api/v1/system \
  word_select_dashboard/server/router/system word_select_dashboard/server/initialize/router.go
git commit -m "feat: expose unified sentence executor config API"
```

Expected: PASS。

## Task 5: 增加 React 统一配置领域逻辑

**Files:**
- Create: `word_select_dashboard/web-react/src/types/executionConfig.ts`
- Create: `word_select_dashboard/web-react/src/lib/executionConfigApi.ts`
- Create: `word_select_dashboard/web-react/src/features/executionConfig.ts`
- Test: `word_select_dashboard/web-react/test/executionConfig.test.ts`

- [ ] **Step 1: 写失败测试：固定模型和当前目标保护**

```ts
test("offers fixed models for codex and gemini", () => {
  assert.deepEqual(CLI_MODEL_OPTIONS.codex.map((item) => item.value), [
    "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.5",
    "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark",
  ]);
  assert.deepEqual(CLI_MODEL_OPTIONS.gemini.map((item) => item.value), ["auto", "pro", "flash", "flash-lite"]);
});

test("cannot delete the active target", () => {
  const config = createDefaultExecutionConfig();
  config.active_target = { type: "cli", id: "codex" };
  assert.equal(canDeleteCLI(config, "codex"), false);
});
```

- [ ] **Step 2: 验证 RED**

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/executionConfig.test.ts
```

Expected: FAIL with module not found。

- [ ] **Step 3: 定义统一类型和 API**

```ts
export type ExecutionTarget = { type: "api"; id: string } | { type: "cli"; id: string };
export type CLIDriver = "codex" | "gemini";
export interface CLIProviderConfigItem {
  id: string; label: string; driver: CLIDriver; command_path: string; model: string;
  reasoning_effort: "" | "low" | "medium" | "high" | "xhigh";
  working_directory: string; timeout_seconds: number; enabled: boolean;
}
export interface ExecutionConfig {
  active_target: ExecutionTarget | null;
  api_providers: AIProviderConfigItem[];
  cli_providers: CLIProviderConfigItem[];
}

export function getExecutionConfig(): Promise<ExecutionConfig> {
  return requestJSON<ExecutionConfig>("/ai/execution-config");
}
export function saveExecutionConfig(config: ExecutionConfig): Promise<ExecutionConfig> {
  return requestJSON<ExecutionConfig>("/ai/execution-config", {method: "POST", body: JSON.stringify(config)});
}
```

- [ ] **Step 4: 实现默认值和完整校验**

`validateExecutionConfig` 检查目标非空且存在、ID 不重复、当前 CLI 已启用、路径/工作目录非空、模型属于驱动白名单、Codex 推理强度合法、Gemini 推理强度为空、超时大于 0。

- [ ] **Step 5: 验证 GREEN、构建并提交**

```bash
node --test --experimental-strip-types test/executionConfig.test.ts
npm run build
git add word_select_dashboard/web-react/src/types/executionConfig.ts \
  word_select_dashboard/web-react/src/lib/executionConfigApi.ts \
  word_select_dashboard/web-react/src/features/executionConfig.ts \
  word_select_dashboard/web-react/test/executionConfig.test.ts
git commit -m "feat: add unified execution config client model"
```

Expected: PASS。

## Task 6: 实现单菜单、双 Tab 和顶部状态栏

**Files:**
- Create: `word_select_dashboard/web-react/src/components/ExecutionConfigPage.tsx`
- Modify: `word_select_dashboard/web-react/src/App.tsx`
- Modify: `word_select_dashboard/web-react/src/styles.css`
- Test: `word_select_dashboard/web-react/test/executionConfigPage.contract.test.ts`

- [ ] **Step 1: 写失败的页面契约测试**

```ts
test("uses one menu with API and CLI tabs", () => {
  const app = readFileSync("src/App.tsx", "utf8");
  const page = readFileSync("src/components/ExecutionConfigPage.tsx", "utf8");
  assert.equal(app.includes("本地 CLI 配置"), false);
  assert.match(page, /模型 API/);
  assert.match(page, /本地 CLI/);
  assert.match(page, /当前造句执行器/);
  assert.match(page, /全局唯一/);
});
```

- [ ] **Step 2: 验证 RED**

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/executionConfigPage.contract.test.ts
```

Expected: FAIL because the component does not exist。

- [ ] **Step 3: 创建共享 draft 和布局 A 状态栏**

```tsx
const configQuery = useQuery({queryKey: ["execution-config"], queryFn: getExecutionConfig});
const [draft, setDraft] = useState<ExecutionConfig | null>(null);
const [tab, setTab] = useState<"api" | "cli">("api");
const [selectedAPI, setSelectedAPI] = useState(0);
const [selectedCLI, setSelectedCLI] = useState(0);
```

顶部状态只通过 `resolveActiveTarget(draft)` 渲染，不能从当前 Tab 或当前编辑项推断。

- [ ] **Step 4: 实现 API 与 CLI Tab**

API “设为当前”写入 `{type:"api", id}`；CLI 写入 `{type:"cli", id}`。删除或禁用当前目标时阻止操作，不自动选择第一项。

驱动切换安全默认值：

```ts
const patch = driver === "codex"
  ? {driver, model: "gpt-5.6-sol", reasoning_effort: "high" as const}
  : {driver, model: "auto", reasoning_effort: "" as const};
```

- [ ] **Step 5: App 只保留一个模型配置菜单**

用 `<ExecutionConfigPage />` 替换旧 AI 页面内容；TTS 模型配置仍是独立菜单，不能合并。

- [ ] **Step 6: 验证 GREEN、全量测试、构建并提交**

```bash
node --test --experimental-strip-types test/*.test.ts
npm run build
git add word_select_dashboard/web-react/src/components/ExecutionConfigPage.tsx \
  word_select_dashboard/web-react/src/App.tsx word_select_dashboard/web-react/src/styles.css \
  word_select_dashboard/web-react/test/executionConfigPage.contract.test.ts
git commit -m "feat: add API and CLI execution tabs"
```

Expected: PASS。

## Task 7: 让 Word Agent 造句读取唯一执行器

**Files:**
- Create: `word_select_dashboard/word-agent/src/word_agent/services/sentence_executor.py`
- Test: `word_select_dashboard/word-agent/tests/test_sentence_executor.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/services/llm_client.py`
- Modify: `word_select_dashboard/word-agent/tests/test_llm_client.py`
- Modify: `word_select_dashboard/word-agent/src/word_agent/core/config.py`

- [ ] **Step 1: 写失败测试：精确读取 singleton 且不回退**

```python
def test_loader_returns_exact_singleton_target(db_conn):
    seed_executor(db_conn, executor_type="cli", executor_id="codex")
    seed_cli(db_conn, provider_id="codex", enabled=True)
    target = SentenceExecutorLoader(settings).load()
    assert (target.type, target.id) == ("cli", "codex")

def test_loader_does_not_fallback_when_target_is_missing(db_conn):
    seed_executor(db_conn, executor_type="cli", executor_id="missing")
    with pytest.raises(LLMConfigError, match="不存在"):
        SentenceExecutorLoader(settings).load()
```

- [ ] **Step 2: 验证 RED**

```bash
cd word_select_dashboard/word-agent
.venv/bin/pytest tests/test_sentence_executor.py -q
```

Expected: FAIL with module not found。

- [ ] **Step 3: 实现目标加载器**

```python
@dataclass(frozen=True)
class SentenceExecutorTarget:
    type: Literal["api", "cli"]
    id: str
    api_provider: AIProvider | None = None

class SentenceExecutorLoader:
    def load(self) -> SentenceExecutorTarget:
        # SELECT singleton_key='default', then exact provider_id in one connection.
        # Raise LLMConfigError for missing, disabled, or incomplete targets.
```

SQL 必须精确读取 singleton 和目标 provider；禁止首行、字母序或 YAML active 回退。

- [ ] **Step 4: 写失败测试：CLI 调用和失败不回退**

```python
@pytest.mark.asyncio
async def test_generate_sentence_uses_cli_runner_for_cli_target(httpx_mock):
    httpx_mock.add_response(
        url="http://host.docker.internal:6018/v1/text/generate",
        json={"content": valid_sentence_json(), "executor_id": "codex", "driver": "codex", "model": "gpt-5.6-sol", "duration_ms": 5},
    )
    result = await client.generate_sentence_from_words(words=["salute"])
    assert result.provider.id == "codex"

@pytest.mark.asyncio
async def test_cli_failure_does_not_call_api(httpx_mock):
    httpx_mock.add_response(status_code=502, json={"detail": "CLI failed"})
    with pytest.raises(LLMRequestError, match="CLI"):
        await client.generate_sentence_from_words(words=["salute"])
    assert len(httpx_mock.get_requests()) == 1
```

- [ ] **Step 5: 抽取统一 Prompt 并接入 Runner**

```python
def build_sentence_prompt(words: list[str]) -> str:
    return (
        "Return one JSON object with exactly sentence, translation_zh, explanation_zh. "
        "Use every requested word in one natural English sentence. Words: " + ", ".join(words)
    )
```

API 继续调用 chat completions；CLI POST `{executor_id, prompt}` 到 `settings.cli_runner_url`，使用 Bearer Token。两条路径最终都调用现有 `_parse_sentence_payload`。

- [ ] **Step 6: 保证评分使用独立 API 模型**

增加测试：CLI 为当前造句目标时，评分仍按 `WORD_AGENT_WORD_CLEAN_SCORE_DEFAULT_MODEL=qwen3.6-flash` 加载 API，不调用 Runner。

- [ ] **Step 7: 验证 GREEN 并提交**

```bash
.venv/bin/pytest tests/test_sentence_executor.py tests/test_llm_client.py tests/test_api.py -q
.venv/bin/ruff check src/word_agent/services/sentence_executor.py \
  src/word_agent/services/llm_client.py tests/test_sentence_executor.py tests/test_llm_client.py
.venv/bin/pytest -q
git add word_select_dashboard/word-agent/src/word_agent/services/sentence_executor.py \
  word_select_dashboard/word-agent/src/word_agent/services/llm_client.py \
  word_select_dashboard/word-agent/src/word_agent/core/config.py \
  word_select_dashboard/word-agent/tests/test_sentence_executor.py \
  word_select_dashboard/word-agent/tests/test_llm_client.py
git commit -m "feat: route sentence generation through one executor"
```

Expected: PASS。

## Task 8: 实现主机 CLI Runner 与 Codex/Gemini 驱动

**Files:**
- Create: `word_select_dashboard/word-agent/src/word_agent/cli_runner/__init__.py`
- Create: `word_select_dashboard/word-agent/src/word_agent/cli_runner/config_client.py`
- Create: `word_select_dashboard/word-agent/src/word_agent/cli_runner/drivers.py`
- Create: `word_select_dashboard/word-agent/src/word_agent/cli_runner/service.py`
- Create: `word_select_dashboard/word-agent/src/word_agent/cli_runner/main.py`
- Test: `word_select_dashboard/word-agent/tests/test_cli_runner_drivers.py`
- Test: `word_select_dashboard/word-agent/tests/test_cli_runner_api.py`

- [ ] **Step 1: 写失败的固定参数测试**

```python
def test_codex_command_is_fixed_and_shell_free(tmp_path):
    invocation = build_codex_invocation(codex_config(), tmp_path / "schema.json", tmp_path / "last.txt")
    assert invocation.argv[:2] == (codex_config().command_path, "exec")
    assert "--ephemeral" in invocation.argv
    assert ("--sandbox", "read-only") == adjacent_pair(invocation.argv, "--sandbox")
    assert invocation.stdin_prompt is True

def test_gemini_command_uses_json_plan_mode():
    invocation = build_gemini_invocation(gemini_config(model="pro"), "hello")
    assert invocation.argv == (
        gemini_config().command_path, "--model", "pro", "--prompt", "hello",
        "--output-format", "json", "--approval-mode", "plan",
    )
```

- [ ] **Step 2: 验证 RED**

```bash
cd word_select_dashboard/word-agent
.venv/bin/pytest tests/test_cli_runner_drivers.py -q
```

Expected: FAIL with module not found。

- [ ] **Step 3: 实现两个固定驱动和输出提取**

`CLIInvocation` 只包含 `argv`、`stdin_text`、`final_output_path`。Codex 固定使用 `exec`、model、reasoning、read-only、ephemeral、output-schema 和 output-last-message；Gemini 固定使用 model、prompt、JSON output、plan approval。

```python
def extract_gemini_content(stdout: str) -> str:
    payload = json.loads(stdout)
    content = payload.get("response") or payload.get("content")
    if not isinstance(content, str) or not content.strip():
        raise CLIRunnerError("Gemini CLI 没有返回文本内容")
    return content.strip()
```

- [ ] **Step 4: 写失败的鉴权、目标和超时测试**

```python
def test_generate_rejects_missing_token(client):
    response = client.post("/v1/text/generate", json={"executor_id": "codex", "prompt": "hello"})
    assert response.status_code == 401

def test_generate_rejects_non_active_cli(client, config_client):
    config_client.active_target = {"type": "cli", "id": "gemini"}
    assert authorized_post(client, executor_id="codex").status_code == 409

@pytest.mark.asyncio
async def test_process_timeout_kills_process(fake_process):
    fake_process.communicate.side_effect = asyncio.TimeoutError
    with pytest.raises(CLIRunnerError, match="超时"):
        await runner.generate(active_codex(), "hello")
    fake_process.kill.assert_called_once()
```

- [ ] **Step 5: 实现配置客户端和子进程生命周期**

Runner GET `http://127.0.0.1:6015/ai/execution-config`，确认请求 ID 与 `active_target` 完全一致。使用 `asyncio.create_subprocess_exec(*argv, shell=False, start_new_session=True)`；超时先终止进程组再 kill。每个 provider 使用容量 1 的 semaphore。

子进程环境移除继承的 `CODEX_*`，再设置 `CODEX_CI=1` 和 `TERM=dumb`；日志只记录 ID、driver、model、耗时和退出码。

- [ ] **Step 6: 实现 FastAPI 入口**

```python
@app.get("/health")
async def health() -> dict[str, str]:
    return {"service": "word-agent-cli-runner", "status": "ok"}

@app.post("/v1/text/generate", response_model=TextGenerationResponse)
async def generate(request: TextGenerationRequest, authorization: str = Header(default="")):
    verify_bearer_token(authorization, settings.cli_runner_token)
    return await runner.generate(request.executor_id, request.prompt)
```

- [ ] **Step 7: 验证 GREEN 并提交**

```bash
.venv/bin/pytest tests/test_cli_runner_drivers.py tests/test_cli_runner_api.py -q
.venv/bin/ruff check src/word_agent/cli_runner tests/test_cli_runner_drivers.py tests/test_cli_runner_api.py
.venv/bin/pytest -q
git add word_select_dashboard/word-agent/src/word_agent/cli_runner \
  word_select_dashboard/word-agent/tests/test_cli_runner_drivers.py \
  word_select_dashboard/word-agent/tests/test_cli_runner_api.py
git commit -m "feat: add authenticated host CLI runner"
```

Expected: PASS，且测试不调用真实模型。

## Task 9: 增加初始化和统一启动/停止脚本

**Files:**
- Create: `scripts/bootstrap_sentence_cli_configs.py`
- Create: `scripts/start_word_select_dashboard.sh`
- Create: `scripts/stop_word_select_dashboard.sh`
- Test: `word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py`
- Modify: `word_select_dashboard/word-agent/.env.example`
- Modify: `.gitignore`

- [ ] **Step 1: 写失败测试：初始化不改变 active target**

```python
def test_bootstrap_adds_two_clis_without_changing_target():
    current = execution_config(active={"type": "api", "id": "aliyun-deepseek"})
    result = add_missing_cli_configs(
        current,
        codex_path="/Applications/ChatGPT.app/Contents/Resources/codex",
        gemini_path="/Users/conchi/.npm-global/bin/gemini",
        working_directory="/Users/conchi/workforce/rob_english_word_workforce",
    )
    assert result["active_target"] == current["active_target"]
    assert [item["id"] for item in result["cli_providers"]] == ["codex", "gemini"]
```

- [ ] **Step 2: 验证 RED**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py -q
```

Expected: FAIL。

- [ ] **Step 3: 实现幂等初始化**

脚本使用 `urllib.request` GET/POST 统一配置。只在 ID 不存在时追加默认 CLI；API Key 保持空字符串以触发后端保留语义；`active_target` 原样回传。

```python
CODEX_CANDIDATES = [shutil.which("codex"), "/Applications/ChatGPT.app/Contents/Resources/codex", "/opt/homebrew/bin/codex"]
GEMINI_CANDIDATES = [shutil.which("gemini"), str(Path.home() / ".npm-global/bin/gemini")]
```

- [ ] **Step 4: 实现统一启动脚本**

```zsh
RUNTIME_DIR="$PROJECT_ROOT/.runtime"
TOKEN_FILE="$RUNTIME_DIR/cli-runner.token"
ENV_FILE="$RUNTIME_DIR/cli-runner.env"
mkdir -p "$RUNTIME_DIR" && chmod 700 "$RUNTIME_DIR"
test -s "$TOKEN_FILE" || openssl rand -hex 32 > "$TOKEN_FILE"
chmod 600 "$TOKEN_FILE"
CLI_RUNNER_TOKEN="$(<"$TOKEN_FILE")"
printf 'WORD_AGENT_CLI_RUNNER_TOKEN=%s\n' "$CLI_RUNNER_TOKEN" > "$ENV_FILE"
chmod 600 "$ENV_FILE"
```

随后使用已有三个本地 Compose 文件启动 6015/6016/6017，执行初始化脚本，以 `.venv/bin/python -m word_agent.cli_runner.main` 启动 6018，并写 PID 文件。

- [ ] **Step 5: 实现只停止当前项目的脚本**

读取 PID，验证命令包含 `word_agent.cli_runner.main` 后 TERM；再对三个明确 Compose 文件执行 `docker compose stop`。不得停止 PostgreSQL、Redis、MinIO 或其他项目。

- [ ] **Step 6: 更新环境示例和忽略规则**

```dotenv
WORD_AGENT_CLI_RUNNER_URL=http://host.docker.internal:6018
WORD_AGENT_CLI_RUNNER_TOKEN=
WORD_AGENT_CLI_RUNNER_CONFIG_URL=http://127.0.0.1:6015/ai/execution-config
WORD_AGENT_CLI_RUNNER_PORT=6018
```

`.gitignore` 增加 `.runtime/` 和 `.superpowers/`。

- [ ] **Step 7: 验证 GREEN 并提交**

```bash
word_select_dashboard/word-agent/.venv/bin/pytest \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py -q
zsh -n scripts/start_word_select_dashboard.sh
zsh -n scripts/stop_word_select_dashboard.sh
git add .gitignore scripts/bootstrap_sentence_cli_configs.py \
  scripts/start_word_select_dashboard.sh scripts/stop_word_select_dashboard.sh \
  word_select_dashboard/word-agent/.env.example \
  word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py
git commit -m "feat: add unified local project startup"
```

Expected: PASS。

## Task 10: 全量回归、Docker 部署和浏览器验收

**Files:**
- Modify only if a verified defect is found in files introduced above.

- [ ] **Step 1: 运行全量验证**

```bash
(cd word_select_dashboard/server && go test ./...)
(cd word_select_dashboard/web-react && node --test --experimental-strip-types test/*.test.ts && npm run build)
(cd word_select_dashboard/word-agent && .venv/bin/pytest -q)
(cd word_select_dashboard/word-agent && .venv/bin/ruff check \
  src/word_agent/services/sentence_executor.py src/word_agent/cli_runner \
  tests/test_sentence_executor.py tests/test_cli_runner_drivers.py tests/test_cli_runner_api.py)
git diff --check
```

Expected: 全部通过；只允许已有 Starlette/httpx deprecation warning。

- [ ] **Step 2: 创建三个当前镜像的回滚标签**

为 Go、React、Word Agent 当前镜像创建 `pre-cli-executor` 标签并记录镜像 ID。任何构建、迁移或健康检查失败时恢复标签并重建对应容器。

- [ ] **Step 3: 从 `dev` 构建并启动**

沿用未跟踪 `deploy/` 中现有 Dockerfile 和 Compose，但不得修改或提交 `deploy/`。运行：

```bash
./scripts/start_word_select_dashboard.sh
```

Expected: 6015、6016、6017、6018 均健康；数据库存在两个新表；原 5 个 API 和 TTS 配置仍存在。

- [ ] **Step 4: 验证脱敏接口**

```bash
curl -fsS http://127.0.0.1:6016/api/ai/execution-config
```

Expected: 5 个 API、Codex、Gemini、唯一 `active_target`；API Key 为空且 `api_key_configured=true`。

- [ ] **Step 5: 浏览器验收布局 A**

确认单一“模型配置”菜单、顶部全局状态栏、API/CLI Tab、固定模型下拉、当前目标禁止禁用/删除、CLI 生效时 API 无勾选；TTS 菜单保持独立。

- [ ] **Step 6: 用假 CLI 完成部署链路测试**

临时将测试 CLI 指向返回固定 JSON 的 fixture，调用真实 `/v1/sentences/generate`，确认 Word Agent/Runner/业务解析连通；测试后恢复原配置。不得调用真实模型。

- [ ] **Step 7: 真实 CLI 调用前再次获取授权**

列出将使用的 Codex/Gemini 模型、Prompt 范围和可能额度。只有用户明确同意后才各执行一次；未获授权时以假 CLI 集成测试作为证据。

- [ ] **Step 8: 最终状态检查**

```bash
git status --short
docker ps --filter name=word-select-dashboard --filter name=word-agent \
  --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
curl -fsS http://127.0.0.1:6015/health
curl -fsS http://127.0.0.1:6017/health
curl -fsS http://127.0.0.1:6018/health
```

Expected: 只剩任务开始前已有的用户未提交改动；四个服务健康。

## 完成定义

- `dev` 包含独立 TTS 实现和 API/CLI 唯一造句执行器。
- Codex、Gemini 和所有 API 共享一个生效选择。
- CLI Runner 属于当前项目，不依赖 `ai-task-center`。
- API/CLI 使用同一业务 JSON 解析，CLI 失败不回退。
- 评分仍使用独立 API 模型，TTS 不受影响。
- 测试、构建、Docker 健康检查和浏览器验收通过。
- 真实模型调用仅在用户明确授权后执行。
