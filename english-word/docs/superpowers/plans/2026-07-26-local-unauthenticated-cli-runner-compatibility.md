# 本地无鉴权 CLI Runner 兼容实施计划

> 设计依据：`docs/superpowers/specs/2026-07-26-local-unauthenticated-cli-runner-compatibility-design.md`

**目标：** 将宿主机 `6018` CLI Runner 恢复为 Word Agent 唯一的 CLI
执行边界，在不使用 Token 的前提下同时兼容原生和 Docker Word Agent。

**架构：** Word Agent 只通过 HTTP `POST /v1/text/generate` 调用 Runner。原生
运行使用 `127.0.0.1:6018`，Docker 使用 `host.docker.internal:6018`。Runner
只读取数据库 DSN 和执行器配置，并在宿主机执行 Codex/Gemini CLI。

**技术栈：** Python 3.12、FastAPI、httpx、pytest、zsh、Docker Compose。

---

## Task 1：恢复 Word Agent 的无鉴权 HTTP CLI 调用

**文件：**

- 修改：`word_select_dashboard/word-agent/tests/test_llm_client.py`
- 修改：`word_select_dashboard/word-agent/src/word_agent/services/llm_client.py`
- 修改：`word_select_dashboard/word-agent/src/word_agent/core/config.py`

1. 修改 CLI 目标测试，要求请求配置的
   `cli_runner_url + /v1/text/generate`，请求体只包含 `executor_id` 和
   `prompt`，不传 `Authorization`。
2. 增加 `select_db_dsn=None` 不影响 CLI HTTP 路径、原生/Docker URL 尾斜线
   规范化、响应执行器不一致和非空内容校验。
3. 运行定向测试，确认旧的进程内 `CLIRunner` 实现按预期失败。
4. 将 `_cli_completion()` 改为复用现有 `http_client.post()` 的 HTTP 调用，
   删除 Word Agent 对 `CLIProviderConfigClient`、`CLIRunner` 和 CLI 数据库
   DSN 的依赖。
5. 删除 `Settings.cli_runner_token`，把默认 Runner URL 改为
   `http://127.0.0.1:6018`。
6. 运行 `tests/test_llm_client.py`，确认 GREEN。

## Task 2：移除 CLI Runner Token 鉴权

**文件：**

- 修改：`word_select_dashboard/word-agent/tests/test_cli_runner_api.py`
- 修改：`word_select_dashboard/word-agent/src/word_agent/cli_runner/service.py`

1. 将 Runner 设置测试改为只要求数据库 DSN；无请求 Header 也允许生成。
2. 删除精确 Bearer Token 和缺 Token 失败的旧断言，保留执行器、配置、驱动、
   超时、脱敏和输出契约覆盖。
3. 运行定向测试，确认旧鉴权实现按预期失败。
4. 从 `CLIRunnerSettings`、环境读取和 FastAPI endpoint 删除 Token 字段、
   Header 依赖和比较逻辑。
5. 运行 `tests/test_cli_runner_api.py`，确认 GREEN。

## Task 3：清理 Bootstrap、Compose 和项目级 Docker 生命周期中的 Token

**文件：**

- 修改：`word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py`
- 修改：`word_select_dashboard/word-agent/scripts/bootstrap_sentence_cli_configs.py`
- 修改：`word_select_dashboard/word-agent/.env.example`
- 修改：`word_select_dashboard/word-agent/docker-compose.yml`
- 修改：`scripts/start_word_select_dashboard.sh`
- 修改：`scripts/stop_word_select_dashboard.sh`

1. 将环境文件测试改为只写宿主机 Runner DSN 和容器 Word Agent DSN，权限保持
   `0600`；删除 Token 文件生成、读取和注入断言。
2. 将 Compose 契约改为固定
   `WORD_AGENT_CLI_RUNNER_URL=http://host.docker.internal:6018`，不要求 Token。
3. 运行定向测试，确认旧脚本和配置按预期失败。
4. 删除 bootstrap 的 `--runner-token-file` 和 Token 参数，保留 DSN 校验及
   原子写入。
5. 删除启动脚本的 OpenSSL/Token 文件逻辑，停止脚本不再注入 Token
   placeholder。
6. 更新 `.env.example`：原生默认 URL 为 `127.0.0.1:6018`；Compose 保持
   Docker Host URL。
7. 运行脚本契约测试，确认 GREEN。

## Task 4：让根目录非 Docker 启动统一管理 6018 Runner

**文件：**

- 修改：`word_select_dashboard/word-agent/tests/test_bootstrap_sentence_cli_configs.py`
- 修改：`restart_all_services.sh`

1. 增加根启动脚本静态契约测试：端口为 `6011`–`6018`、Runner 在 Word Agent
   前启动、Word Agent 原生 URL 指向 `127.0.0.1:6018`、URL 展示与等待端口正确。
2. 运行新测试，确认当前旧端口和缺 Runner 的脚本按预期失败。
3. 在根脚本中通过 bootstrap 生成受限权限的 Runner 环境文件，以独立进程启动
   `word_agent.cli_runner.main`，等待 `6018` 后再启动 Word Agent。
4. 更新所有本地服务端口、Java 到 Word Agent 的地址、状态和 URL 输出；不启动
   Docker 应用容器。
5. 运行脚本契约测试和 `zsh -n restart_all_services.sh`，确认 GREEN。

## Task 5：回归测试与配置验证

1. 在 `word_select_dashboard/word-agent` 运行：
   `./.venv/bin/pytest -q tests/test_llm_client.py tests/test_cli_runner_api.py tests/test_bootstrap_sentence_cli_configs.py`。
2. 运行 Word Agent 完整测试：`./.venv/bin/pytest -q`。
3. 运行 `docker compose -f docker-compose.yml config`，确认 Compose 可解析且无
   Runner Token 变量。
4. 使用 `rg` 确认生产运行链路不再引用
   `WORD_AGENT_CLI_RUNNER_TOKEN` 或 `cli_runner_token`。

## Task 6：非 Docker 真实链路自测

1. 使用根启动脚本重启非 Docker 项目，确认 `6011`–`6018` 都由当前项目进程
   监听。
2. 请求 Runner `/health` 和 Word Agent `/health`。
3. 使用不在错题表中的合成单词调用 Word Agent 造句接口，验证请求经过
   `6018`、Codex CLI 返回合法句子 JSON。
4. 只读查询错题状态计数，确认测试前后 `wrong_word_events` 的
   `failed/retry_wait` 数量和事件内容未被测试修改。
5. 汇总测试命令、结果和未覆盖的运行条件。
