---
title: 单词选择 Go 管理服务
summary: 为运营管理端提供用户、词库、句子、完形结果、执行记录及 AI/TTS 配置 API，并编排 Python Agent。
---

# 单词选择 Go 管理服务

## 职责与边界

- 为 React 运营管理端提供登录、用户学习数据、词库、清洗单词、句子和完形结果 API。
- 保存 AI Provider、执行和 TTS 配置，管理运行记录与工作流状态。
- 通过 HTTP 调用 Python Agent 执行句子生成和句子评分等能力。
- 不承担用户实时对战；用户侧游戏和完形提交由 Java 后端负责。

## 项目定位

| 项目 | 值 |
| --- | --- |
| 相对路径 | `word_select_dashboard/server` |
| 技术栈 | Go 1.25、Gin、Gorm、Viper、PostgreSQL、Redis、MinIO |
| 启动入口 | `main.go` |
| 路由装配 | `initialize/router.go`、`router/system/` |
| 主要配置 | `config.yaml`、`config.docker.yaml`、`config/` |

## 入站与出站

- 后台鉴权与用户：`base/login`、`user/**`、`users/**`。
- 词库：`word-libraries/**`，包括词库、清洗单词、句子和评分入口。
- 句子：`sentences/generate`、`sentences/history`。
- 完形结果：`cloze-results/users`、`cloze-results/items`。
- 配置：`ai/config`、`ai/execution-config`、`tts/config`。
- 执行记录：`executions/runs`。
- 对 Python Agent：`/v1/sentences/generate`、`/v1/word-clean-sentences/score`；目标地址由配置提供。
- 对对象存储：通过 MinIO Client 管理或暴露音频文件；具体 Bucket 和凭据不进入文档。

## 配置与数据源

- 主要管理库为 `select_english_word`，保存后台账号、AI/TTS 配置、执行记录、词库和句子数据。
- 同时读取 `rob_english_word` 的用户学习、错词、掌握词、训练和完形结果。
- 数据库驱动层支持多种 Engine，但本工作空间当前业务映射以 PostgreSQL 为准。
- `System.RouterPrefix` 决定 API 前缀；前端源码以同源 `/api/**` 调用为主。

## 关键代码锚点

- 启动与装配：`main.go`、`initialize/router.go`
- 路由：`router/system/`
- API：`api/v1/system/`
- 业务服务：`service/system/`
- 模型：`model/system/`
- 句子调用 Agent：`api/v1/system/sys_sentence.go`
- 清洗句子评分：`api/v1/system/sys_word_library.go`
- AI、执行和 TTS 配置：`service/system/sys_ai_config.go`、`sys_execution_config.go`、`sys_tts_config.go`

## 关联链路

- [AI 句子生成、评分与 TTS](../../chains/ai-sentence-tts-scoring.md)
- [错词与掌握进度](../../chains/wrong-word-mastery.md)
- [完形练习与复习](../../chains/cloze-practice-review.md)

## 运行时待核对

- React 静态资源由 Go 服务托管还是由独立 Nginx 容器提供，以当前部署模式为准。
- Python Agent 超时、重试和部分成功时，执行记录的最终状态一致性。
- 两个逻辑库跨库读取的账号权限和事务边界。

## 复核信息

- 复核日期：2026-08-01；源码基线：`dev` 分支。
- 证据：`go.mod`、`main.go`、Router、API、Service、Model 和部署配置。
