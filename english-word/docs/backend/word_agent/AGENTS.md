---
title: 单词智能代理 Python 服务
summary: 执行句子生成、清洗句子评分、错词任务和 MiMo TTS，并与 Go 服务、数据库和对象存储协作。
---

# 单词智能代理 Python 服务

## 职责与边界

- 提供句子生成、TTS、错词事件和清洗句子评分 HTTP 能力。
- 调用配置的 LLM Provider 与 MiMo TTS，必要时把音频写入对象存储。
- 通过 PostgreSQL 读取或更新执行任务、句子、评分及错词相关数据。
- CLI Runner 提供受控的本地任务执行入口；运营编排和配置管理仍由 Go 服务负责。

## 项目定位

| 项目 | 值 |
| --- | --- |
| 相对路径 | `word_select_dashboard/word-agent` |
| 技术栈 | Python 3.12–3.14、FastAPI、HTTPX、OpenAI SDK、psycopg、MinIO |
| 包入口 | `src/word_agent/main.py` |
| API 路由 | `src/word_agent/api/routes.py` |
| CLI Runner | `src/word_agent/cli_runner/` |
| 构建配置 | `pyproject.toml` |

## HTTP 能力

- `GET /health`：健康检查。
- `POST /v1/sentences/generate`：根据词集合生成英文句子。
- `POST /v1/tts/generate`、`GET /v1/tts/files/{file_name}`：生成和读取 TTS 结果。
- `POST /v1/wrong-words/events`：处理错词相关事件。
- `POST /v1/word-clean-sentences/score`：批量评价清洗句子。

## 服务与数据

- `services/llm_client.py`：模型请求、句子生成和评分。
- `services/mimo_tts.py`：MiMo TTS 调用与音频处理。
- `services/wrong_word_strategy.py`：错词任务选择和处理策略。
- `services/word_clean_sentence_score.py`：清洗句子评分编排。
- 逻辑数据范围覆盖 `select_english_word` 和 `rob_english_word`；只记录用途，不在文档保存连接信息。
- 外部 Provider、TTS、MinIO 的地址与凭据来自运行配置或环境变量。

## 验证方式

- 单元测试：`pytest`，测试目录 `tests/`。
- 静态检查：Ruff，配置位于 `pyproject.toml`。
- 服务健康检查不能替代模型、TTS、数据库和对象存储的端到端验证。

## 关联链路

- [AI 句子生成、评分与 TTS](../../chains/ai-sentence-tts-scoring.md)
- [错词与掌握进度](../../chains/wrong-word-mastery.md)

## 运行时待核对

- Provider 限流、超时、重试和输出格式异常的生产表现。
- TTS 生成成功但对象存储写入失败时的补偿策略。
- 错词事件的触发频率、并发锁和重复消费处理。

## 复核信息

- 复核日期：2026-08-01；源码基线：`dev` 分支。
- 证据：`pyproject.toml`、FastAPI 路由、Service、CLI Runner、测试和部署文件。
