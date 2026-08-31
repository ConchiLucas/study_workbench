# API / CLI 统一造句执行器设计

## 背景

当前 Word Agent 的造句链路只能从 `ai_provider_configs` 中选择一个 OpenAI Compatible 模型 API。用户希望在现有“模型配置”菜单中加入本地 CLI 配置，通过 Tab 区分模型 API 与本地 CLI，并保证造句时全局只有一个执行器生效。

当前项目使用 Docker 部署，而 Codex CLI 和 Gemini CLI 安装在 macOS 主机上。Linux 容器不能直接执行 macOS CLI，因此需要在当前项目内增加一个主机侧 CLI Runner。该 Runner 不依赖 `ai-task-center`，两个项目继续保持完全独立。

## 已确认需求

- 左侧只保留一个“模型配置”菜单。
- 页面使用“模型 API”和“本地 CLI”两个 Tab。
- 顶部始终显示唯一的“当前造句执行器”。
- API 模型、Codex CLI、Gemini CLI 之间全局只能选择一个作为造句执行器。
- 选择 CLI 后不保留上一次 API 选择；切回 API 时必须重新选一个 API 模型。
- 本次选择只影响错词造句和普通句子生成，不改变例句评分等其他 AI 功能。
- CLI 与 API 具有相同的业务语义：发送文本 Prompt，返回文本内容，再由 Word Agent 解析为句子、翻译和说明。
- 当前支持 Codex 与 Gemini 两种固定驱动；未来增加其他 CLI 时新增驱动实现，不开放任意参数模板。
- CLI 模型使用固定下拉列表。
- CLI 失败时不自动回退到其他 CLI 或 API。
- 使用项目统一启动脚本启动主机 CLI Runner 和现有 Docker 服务，不安装 macOS 开机启动项。

## 非目标

- 不把当前项目与 `ai-task-center` 合并或建立运行时依赖。
- 不让 CLI 承担例句评分、TTS 或代码修改任务。
- 不开放用户自定义 shell 参数、命令模板或任意 HTTP 命令执行。
- 不实现 CLI 自动回退、负载均衡或多执行器并发选择。
- 不在本次修改中重构无关的 AI、TTS 或用户业务页面。

## 用户界面

“模型配置”页面采用已确认的布局 A。

### 顶部状态栏

状态栏在两个 Tab 上方固定显示，格式示例：

- `当前造句执行器：Aliyun DeepSeek V3.2 · deepseek-v3.2`
- `当前造句执行器：Codex CLI · 5.6 Sol · High`
- `当前造句执行器：Gemini CLI · Pro`

状态栏同时显示“全局唯一”标识。当前执行器不可用时显示明确的异常状态，而不是自动选择其他配置。

### 模型 API Tab

保留现有 API 配置列表和编辑表单。列表中的勾选标记只依据统一造句执行器状态显示，不再仅依据旧的 `ai_provider_configs.active` 推断。

字段保持为：

- 配置 ID
- 显示名称
- 接口类型
- Base URL
- API Key
- 模型名称
- Max Tokens

API Key 在读取接口中脱敏；空值保存表示保留数据库中的现有值。

### 本地 CLI Tab

CLI 配置列表与 API 列表使用相同的视觉结构，显示名称、驱动类型、模型摘要以及当前生效标记。

字段包括：

- 配置 ID
- 显示名称
- CLI 驱动：`codex` 或 `gemini`
- 命令路径
- 模型
- Codex 推理强度
- 工作目录
- 超时时间（秒）
- 启用状态

驱动切换后，页面使用该驱动对应的固定模型选项并执行字段校验。Gemini 不显示推理强度字段。

Codex 模型选项：

- `gpt-5.6-sol`
- `gpt-5.6-terra`
- `gpt-5.6-luna`
- `gpt-5.5`
- `gpt-5.4`
- `gpt-5.4-mini`
- `gpt-5.3-codex-spark`

Codex 推理强度选项：`low`、`medium`、`high`、`xhigh`。

Gemini 模型选项使用官方稳定别名：`auto`、`pro`、`flash`、`flash-lite`。

初始 CLI 配置使用本机已检测到的路径：

- Codex：`/Applications/ChatGPT.app/Contents/Resources/codex`
- Gemini：`/Users/conchi/.npm-global/bin/gemini`

CLI 配置支持新增和删除，但驱动类型只能选择已实现的驱动。删除或禁用当前造句执行器前必须先选择其他执行器。

## 数据模型

### 现有 API 配置

继续使用 `ai_provider_configs`。为兼容旧代码，切换到 API 执行器时同步维护该表的 `active` 字段；切换到 CLI 时将所有 API 行的 `active` 设为 `false`。

API 配置本身不会因切换执行器而删除。

### CLI 配置表

新增 `cli_provider_configs`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint | 主键 |
| `provider_id` | varchar，唯一 | CLI 配置 ID |
| `label` | varchar | 显示名称 |
| `driver` | varchar | `codex` 或 `gemini` |
| `command_path` | text | 主机命令绝对路径 |
| `model` | varchar | 固定下拉中的模型值 |
| `reasoning_effort` | varchar | Codex 推理强度；Gemini 为空 |
| `working_directory` | text | 主机工作目录 |
| `timeout_seconds` | integer | 执行超时 |
| `enabled` | boolean | 是否可选 |
| `created_at` | timestamptz | 创建时间 |
| `updated_at` | timestamptz | 更新时间 |

CLI 表不单独保存 `active`；唯一生效状态由统一执行器表维护，避免出现两个事实来源。

### 唯一造句执行器表

新增单行表 `sentence_executor_config`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `singleton_key` | varchar，主键 | 固定为 `default` |
| `executor_type` | varchar | `api` 或 `cli` |
| `executor_id` | varchar | API 或 CLI 的 `provider_id` |
| `updated_at` | timestamptz | 更新时间 |

后端在事务中验证目标存在且可用。该单行结构从数据层保证造句执行器只有一个。

## 管理 API

新增统一配置接口：

- `GET /ai/execution-config`
- `POST /ai/execution-config`

返回结构包含：

```json
{
  "active_target": { "type": "cli", "id": "codex" },
  "api_providers": [],
  "cli_providers": []
}
```

POST 在同一个数据库事务内完成：

1. 校验 API 和 CLI 配置。
2. 校验 `active_target` 存在且已启用。
3. 保存 API 配置，空 API Key 保留原值。
4. 保存 CLI 配置。
5. 写入唯一造句执行器。
6. 同步旧 `ai_provider_configs.active` 兼容字段。

数据库是造句执行器的唯一事实来源。现有 `/ai/config` 接口保留用于向后兼容；新页面改用统一接口。Go 内存和 `config.yaml` 中的 API 配置在数据库事务成功后尽力同步，但不再决定造句执行器。

## 运行时架构

### Word Agent

`/v1/sentences/generate` 每次请求都从数据库读取 `sentence_executor_config`，不缓存选择，从而让后台保存后下一次造句立即生效。

Word Agent 引入统一文本执行器接口：

```text
generate(prompt) -> raw_text, executor_metadata
```

- API 适配器沿用现有 HTTP 模型调用。
- CLI 适配器调用主机 CLI Runner。
- 两种适配器返回相同的原始文本抽象。
- 现有句子 JSON 解析器统一处理两种返回结果。

例句评分继续使用独立的 `WORD_AGENT_WORD_CLEAN_SCORE_DEFAULT_MODEL`，不读取 `sentence_executor_config`。

### 主机 CLI Runner

Runner 属于当前仓库的 Word Agent 子项目，但作为 macOS 主机进程运行。它提供：

- `GET /health`
- `POST /v1/text/generate`

Word Agent 容器通过 `http://host.docker.internal:6018` 调用。请求只携带当前 CLI 配置 ID 和造句 Prompt，不能指定命令、参数或工作目录。Runner 通过主机可访问的 Go 管理接口读取最新统一配置，确认请求 ID 等于当前生效的 CLI，并从返回的 CLI 配置中取得驱动、命令路径、模型和运行参数；校验不一致时拒绝执行。

Runner 返回统一结构：

```json
{
  "content": "模型最终文本",
  "executor_id": "codex",
  "driver": "codex",
  "model": "gpt-5.6-sol",
  "duration_ms": 1234
}
```

### Codex 驱动

Codex 使用参数数组直接启动子进程，不经过 shell。固定行为包括：

- `exec`
- 指定 `--model`
- 指定 `model_reasoning_effort`
- `--sandbox read-only`
- `--ephemeral`
- 固定句子 JSON Schema
- Prompt 通过标准输入传递
- 最终消息写入临时文件并由 Runner 读取

### Gemini 驱动

Gemini 同样使用参数数组直接启动。固定行为包括：

- `--prompt`
- 指定 `--model`
- `--output-format json`
- `--approval-mode plan`
- Runner 从 Gemini JSON 外层结构中提取最终文本

### 输出契约

两种 CLI 与 API 使用相同的造句指令，最终内容必须是：

```json
{
  "sentence": "...",
  "translation_zh": "...",
  "explanation_zh": "..."
}
```

Word Agent 继续负责去除代码围栏、解析 JSON、验证三个字段以及写入后续 TTS/MinIO 流程。CLI Runner 不包含业务数据库写入逻辑。

## 启动与停止

新增统一启动脚本，顺序为：

1. 检查 PostgreSQL、Docker 和两个 CLI 命令路径。
2. 生成或读取本地 Runner 共享 Token，并写入未提交的 `.runtime/cli-runner.env`。
3. 使用该环境文件启动 Go 后端、React 前端和 Word Agent Docker 服务。
4. 使用同一 Token 启动主机 CLI Runner。
5. 检查 6015、6016、6017、6018 健康接口。

同时提供停止脚本，只停止当前项目的 CLI Runner 和三个应用容器，不影响 PostgreSQL、Redis、MinIO 或其他项目。

CLI Runner PID、日志和自动生成的 Token 放在未提交的 `.runtime/` 目录。

## 安全设计

- Runner 监听 `0.0.0.0:6018` 以允许 Docker Desktop 通过 `host.docker.internal` 访问，并要求 Bearer Token。
- Token 首次启动时随机生成，文件权限限制为当前用户可读。
- Word Agent Docker 通过环境变量接收同一个 Token。
- Runner 不接受命令路径、参数数组或工作目录作为 HTTP 请求字段。
- 命令路径、工作目录和模型只从已保存配置读取。
- 子进程使用参数数组，不执行 `shell=True`。
- 子进程继承环境时移除内部 `CODEX_*` 沙箱变量，但保留正常 CLI 登录所需的用户环境。
- 日志不记录 Token、API Key、完整 Prompt 或完整模型输出。
- 错误响应只返回截断、脱敏后的 stderr 摘要。
- 每个 CLI 配置限制并发子进程数，默认同一配置一次只执行一个请求。

## 错误处理

以下情况直接返回明确错误，不回退：

- 未选择造句执行器。
- 目标配置不存在或已禁用。
- CLI Runner 未启动或 Token 不匹配。
- CLI 命令路径不存在或不可执行。
- CLI 登录失效或模型不可用。
- 子进程超时、异常退出或没有输出。
- CLI 外层 JSON 无法解析。
- 最终句子 JSON 缺少业务字段。

Go 页面保存错误、Word Agent 业务错误和 CLI Runner 执行错误使用不同错误码/消息，方便定位故障层级。

## 迁移与兼容

- 数据库自动迁移创建两个新表。
- 一次性初始化脚本检测本机 Codex/Gemini 路径，并在 CLI 表为空时写入两个默认配置。
- 初始化不改变当前 API 执行器。
- 当前 `ai_provider_configs` 五个模型配置保留。
- 当前 TTS 配置表和 TTS 运行链路不变。
- 旧 `/ai/config` 接口保留，避免已有调用方立即失效。
- Word Agent 不再使用 YAML 的 `ai.active` 决定造句执行器，但其他兼容代码可继续读取 API 配置。

## 测试策略

### Go

- 数据库模型和自动迁移测试。
- API/CLI 配置规范化测试。
- 唯一造句执行器校验测试。
- API 与 CLI 切换时旧 `active` 字段同步测试。
- 保存事务失败回滚测试。
- API Key 脱敏和空值保留测试。
- 管理 API 路由测试。

### React

- 单一菜单和两个 Tab 的契约测试。
- 顶部全局状态栏测试。
- API/CLI 全局互斥选择测试。
- Codex/Gemini 驱动字段切换测试。
- 固定模型选项和 Codex 推理强度测试。
- 禁用/删除当前执行器的防护测试。

### Word Agent 与 CLI Runner

- 统一执行器解析测试。
- API 执行器回归测试。
- Codex 驱动参数、标准输入、Schema、超时和退出码测试。
- Gemini 驱动参数、JSON 外层提取、超时和退出码测试。
- Runner Token 鉴权测试。
- 使用假的 CLI 可执行文件完成主机 Runner 集成测试，不调用真实模型。
- API、Codex、Gemini 三种模拟返回都通过同一业务 JSON 解析测试。

### 部署验收

- 统一启动脚本成功启动 6015、6016、6017、6018。
- 浏览器确认布局 A、两个 Tab 和唯一状态标记。
- 数据库确认唯一执行器行和五个原 API 配置未丢失。
- 经用户明确同意后，分别执行一次真实 Codex/Gemini 造句测试。

## 验收标准

- 后台只显示一个“模型配置”菜单，并通过 Tab 管理 API 与 CLI。
- 任意时刻页面和数据库都只有一个造句执行器生效。
- Codex CLI、Gemini CLI 和任意 API 模型都可以被设为唯一造句执行器。
- CLI 与 API 对 Word Agent 暴露一致的文本输入/文本输出行为。
- 切换后下一次造句立即使用新执行器。
- CLI 故障不会静默使用其他模型。
- 例句评分、TTS 和原有五个 API 配置不受破坏。
