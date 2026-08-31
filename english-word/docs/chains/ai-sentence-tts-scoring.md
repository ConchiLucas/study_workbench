---
title: 英语 AI 句子生成、评分与 TTS 链路
summary: 说明运营后台通过 Go 服务调用 Python Agent，再访问模型、TTS、对象存储和数据库的端到端流程。
---

# 英语 AI 句子生成、评分与 TTS 链路

## 主路径

```text
React 运营管理端
  -> Go /api/sentences/generate 或 word-libraries 评分入口
  -> Go 保存/更新执行记录
  -> HTTP 调用 Python Agent
  -> /v1/sentences/generate | /v1/word-clean-sentences/score | /v1/tts/generate
  -> LLM Provider 或 MiMo TTS
  -> 结果校验与业务转换
  -> PostgreSQL 回写句子、评分和执行状态
  -> 可选写入 MinIO 音频
  -> Go 返回或后续查询
  -> React 展示结果与状态
```

## 配置流

- React 管理端通过 Go API维护 AI、执行和 TTS 配置。
- Go 是配置发布和任务编排方；Python Agent 是实际 Provider 调用方。
- 密钥、Endpoint 和对象存储凭据只从受控配置加载，不写入文档或普通日志。

## 失败边界

- Go 创建任务失败：不应调用 Agent。
- Agent 网络或模型失败：执行记录应保存可诊断但脱敏的失败状态。
- 生成成功但数据库回写失败：不能向前端报告完整成功。
- TTS 成功但对象存储失败：保留可重试状态，不能留下不可访问的完成记录。
- 前端超时不代表服务端任务一定失败，应通过执行记录查询最终状态。

## 证据路径

- `word_select_dashboard/web-react/src/lib/sentenceApi.ts`
- `word_select_dashboard/web-react/src/lib/wordLibraryApi.ts`
- `word_select_dashboard/web-react/src/lib/executionApi.ts`
- `word_select_dashboard/server/api/v1/system/sys_sentence.go`
- `word_select_dashboard/server/api/v1/system/sys_word_library.go`
- `word_select_dashboard/word-agent/src/word_agent/api/routes.py`
- `word_select_dashboard/word-agent/src/word_agent/services/llm_client.py`
- `word_select_dashboard/word-agent/src/word_agent/services/mimo_tts.py`

## 运行时待核对

- Provider 限流、重试退避、任务并发和请求取消传播。
- 长任务是否同步等待、异步轮询或由 CLI Runner 执行，以当前配置和入口为准。
