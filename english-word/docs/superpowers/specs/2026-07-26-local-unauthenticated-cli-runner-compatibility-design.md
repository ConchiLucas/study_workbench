# 本地无鉴权 CLI Runner 的 Docker / 非 Docker 兼容设计

## 背景

当前造句执行器选择为 `cli/codex`。Codex CLI 本身、ChatGPT 登录、模型
`gpt-5.3-codex-spark` 和结构化输出已经通过真实合成词生成验证。

当前失败发生在 Word Agent 的 CLI 接入层：提交 `61d4f72` 将原本的宿主机
HTTP CLI Runner 调用改为 Word Agent 进程内直接构造 `CLIRunner`，并在
`select_db_dsn` 为 `None` 时直接调用 `.strip()`。非 Docker 启动因此在调用
Codex CLI 前返回 HTTP 500；Docker 容器即使修复该空值问题，也不能直接执行
macOS 的 Codex 二进制。

本设计恢复“宿主机 CLI Runner 是唯一 CLI 执行边界”，让 Docker 和非 Docker
Word Agent 复用同一套 HTTP 调用代码。按用户确认，本地开发环境暂不使用 Token
鉴权。

## 已确认需求

- 同时兼容 Docker 和非 Docker 启动。
- Codex CLI 只在宿主机运行，不安装到 Word Agent 容器。
- Word Agent 统一通过宿主机 `6018` CLI Runner 执行 CLI。
- 非 Docker 使用 `127.0.0.1:6018`。
- Docker 使用 `host.docker.internal:6018`。
- 当前仅供个人本地使用，`6018` 暂不使用 Token 或其他鉴权。
- API 模型和 CLI 模型的统一执行器选择、管理页面、数据库表保持不变。
- CLI 失败时不自动回退到 API 模型或其他 CLI。
- 不在代码验证阶段修改或重置现有错词事件状态。

## 非目标

- 不实现公网、共享开发机或局域网多用户场景的安全加固。
- 不在 Docker 镜像中安装或登录 Codex/Gemini CLI。
- 不重新设计管理后台的模型配置页面。
- 不修改 API 模型、TTS、MinIO 或 Java 句子落库的业务契约。
- 不在本次代码修复中自动重置 18 条 `failed` 事件。
- 不开放任意命令、任意参数或任意工作目录的 HTTP 执行能力。

## 架构

```text
非 Docker Word Agent 6017
        │
        │ http://127.0.0.1:6018
        ▼
宿主机 CLI Runner 6018 ── Codex / Gemini CLI
        ▲
        │ http://host.docker.internal:6018
        │
Docker Word Agent 6017
```

两种 Word Agent 运行方式只改变 `WORD_AGENT_CLI_RUNNER_URL`，不改变业务代码。
Runner 从 `select_english_word` 数据库读取当前生效的 CLI 配置，验证请求中的
`executor_id` 与 `sentence_executor_config` 一致，然后按固定驱动规则执行 CLI。

## 组件设计

### Word Agent CLI 适配器

`LLMClient._cli_completion()` 恢复为 HTTP 客户端：

1. 从 `settings.cli_runner_url` 读取 Runner 地址。
2. 向 `/v1/text/generate` 发送 `executor_id` 和造句 Prompt。
3. 不发送 `Authorization` Header。
4. 校验 HTTP 状态、JSON 顶层结构、返回执行器 ID 和非空 `content`。
5. 将 `content` 交给现有句子 JSON 解析器。

CLI 适配器不再：

- 读取 `select_db_dsn`。
- 读取 `WORD_AGENT_CLI_RUNNER_DB_DSN`。
- 直接构造 `CLIProviderConfigClient` 或 `CLIRunner`。
- 在 Word Agent 进程内启动 CLI 子进程。

这样可以直接消除当前 `None.strip()`，并保证容器和宿主机使用完全相同的
Word Agent 代码路径。

### 宿主机 CLI Runner

Runner 保留：

- `GET /health`
- `POST /v1/text/generate`
- 数据库中的当前执行器和 CLI 配置校验
- Codex/Gemini 固定驱动
- 子进程超时、进程组终止和结构化输出读取
- 每个执行器单并发限制

Runner 删除：

- `WORD_AGENT_CLI_RUNNER_TOKEN` 必填检查。
- `Authorization: Bearer ...` Header 解析。
- Token 比较和 401 响应。

`CLIRunnerSettings` 只要求数据库 DSN 完整。`/health` 在数据库 DSN 未配置时
返回不可用；不实际调用 CLI，避免健康检查消耗模型额度。

Runner 请求仍然只能提供 `executor_id` 和 `prompt`。命令路径、模型、推理强度、
工作目录和超时继续只从数据库读取，不能由 HTTP 调用方覆盖。

Runner 监听 `0.0.0.0:6018`，以便宿主机和 Docker 都能访问；该监听方式只允许
用于下文定义的个人本地环境。

### 配置

Word Agent 保留一个运行时配置：

```text
WORD_AGENT_CLI_RUNNER_URL
```

推荐值：

| 运行方式 | URL |
| --- | --- |
| 非 Docker | `http://127.0.0.1:6018` |
| Docker Desktop | `http://host.docker.internal:6018` |

`Settings.cli_runner_token`、`WORD_AGENT_CLI_RUNNER_TOKEN` 和 Compose 中对应环境
变量从当前运行链路删除。

Runner 自己继续读取：

```text
WORD_AGENT_CLI_RUNNER_DB_DSN
```

该 DSN 仅供宿主机 Runner 查询执行器配置。包含数据库密码的运行时环境文件继续
使用当前用户可读权限，且不得提交 Git 或输出到日志。

Docker Compose 显式配置 `host.docker.internal`；在需要兼容 Linux Docker 时使用
`host-gateway` 映射，Docker Desktop for Mac 继续使用内置解析。

## 启动与停止

### 非 Docker

统一宿主机启动流程增加 CLI Runner：

1. 验证数据库和 CLI 命令路径。
2. 生成只包含 Runner DSN 的本地运行时环境文件。
3. 启动宿主机 CLI Runner 6018。
4. 等待 `/health` 返回 200。
5. 启动 Word Agent 6017 及其余前后端服务。
6. 验证 6011–6018 的对应进程和健康状态。

根启动脚本需要把 Runner 作为独立服务管理，而不是只启动现有六个应用项目。
停止流程只停止当前项目的 Runner，不影响其他项目或基础设施。

### Docker

Docker 启动流程为：

1. 在宿主机启动 CLI Runner 6018。
2. 确认宿主机 Runner 健康。
3. 启动 Word Agent 容器，并显式注入
   `WORD_AGENT_CLI_RUNNER_URL=http://host.docker.internal:6018`。
4. 从容器内验证 Runner 可达后，再接受造句任务。

Runner 不进入 Compose，不在容器中复制 macOS CLI 或 ChatGPT 登录文件。

## 错误处理

Word Agent 对 CLI 路径使用明确错误分类：

- Runner URL 缺失：配置错误。
- Runner 连接失败或超时：Runner 不可达。
- Runner 409：当前执行器或 CLI 配置不一致。
- Runner 502：CLI 命令、登录、模型、超时或输出失败。
- Runner 其他非 2xx：Runner 请求失败。
- 返回 JSON 缺字段或执行器 ID 不一致：Runner 返回契约错误。
- CLI 最终内容不是合法句子 JSON：造句结果格式错误。

上述错误继续交给现有错词队列重试状态机处理，不自动切换执行器。日志记录
执行器 ID、状态码和耗时，不记录完整 Prompt、完整模型输出、数据库 DSN 或其他
凭证。

## 本地无鉴权边界

Runner 为了让 Docker 访问，需要监听宿主机可达地址。取消鉴权意味着任何能够
访问宿主机 6018 的本地进程、容器或局域网设备都可能请求 CLI。

本设计接受这一风险，但明确限制为：

- 只在用户个人本机运行。
- 不做端口转发。
- 不部署到公网或共享服务器。
- 不在路由器、防火墙或云安全组中开放 6018。
- FastAPI 文档入口继续关闭。
- HTTP 请求仍不能指定命令路径或任意 CLI 参数。

未来需要共享部署时，应恢复 Token、mTLS 或 Unix Socket/受控代理；本次不实现。

## 测试策略

### Word Agent

- CLI 目标通过配置的 Runner URL 请求 `/v1/text/generate`。
- 请求只包含 `executor_id` 和 `prompt`，不包含 Authorization Header。
- 非 Docker URL 和 Docker URL 都能正确规范化。
- Runner 连接失败、超时、409、502 和畸形响应转换为明确业务错误。
- CLI 失败不调用 API 模型。
- `Settings(select_db_dsn=None)` 不再影响 CLI HTTP 路径。
- 删除或改写仍断言旧实现行为的测试，保证测试与生产代码一致。

### CLI Runner

- 只配置数据库 DSN 时 `/health` 返回 200。
- `/v1/text/generate` 无 Authorization Header 也能执行。
- 执行器 ID 不一致、配置缺失或禁用时拒绝。
- Codex/Gemini 参数数组、安全工作目录、超时和进程组终止保持覆盖。
- 输出契约、错误脱敏和日志不记录 Prompt 保持覆盖。

### 启动脚本和 Compose

- 不再生成、读取或传递 Runner Token。
- Runner 环境文件只保存数据库 DSN 并保持受限权限。
- 非 Docker Word Agent 使用 `127.0.0.1:6018`。
- Docker Word Agent 使用 `host.docker.internal:6018`。
- 启动失败时能区分 Runner 未启动和 Word Agent 未启动。
- 停止脚本只结束当前项目 Runner。

### 运行态验收

1. 验证 Codex CLI 版本和登录状态。
2. 使用无用户数据的合成词从 Runner 直接生成结构化内容。
3. 非 Docker 模式通过 Word Agent `/v1/sentences/generate` 完成文本、TTS 和
   MinIO 全链路。
4. Docker 模式用相同合成词完成同一全链路。
5. 两种模式都确认 CLI Runner 日志出现一次受控执行，Word Agent 不出现
   `None.strip()`。

运行态验收只使用合成词，不修改 `wrong_word_events`。现有 `retry_wait` 和
`failed` 事件的恢复属于代码验证后的独立操作；执行前重新确认目标批次和用户
授权。

## 迁移与回滚

### 迁移

- 不修改数据库表或当前执行器数据。
- 删除旧 Token 运行时文件只属于本机启动清理，不进入数据库迁移。
- 先完成单元测试和合成词 Runner 验证，再重启应用。
- 非 Docker 和 Docker 分别验收后，才讨论历史队列恢复。

### 回滚

- 回滚 Word Agent CLI 适配器和 Runner 无鉴权变更。
- 恢复原 Runner Token 环境变量和启动脚本。
- 不需要回滚数据库。
- 回滚不改变错词事件状态或已生成句子。

## 验收标准

- Codex CLI 保持可用，结构化生成成功。
- 非 Docker Word Agent 不再出现 `select_db_dsn=None` 的 `.strip()` 异常。
- Docker 和非 Docker Word Agent 使用同一个 HTTP CLI 适配器。
- 两种模式都能通过宿主机 6018 Runner 完成合成词句子生成。
- Runner 接口按用户要求不需要 Token。
- Runner 不接受任意命令、参数或工作目录。
- API 模型、TTS、MinIO、管理后台和数据库执行器选择不发生无关变化。
- 代码验证阶段不修改现有错词队列状态。
