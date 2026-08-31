---
title: 英语单词系统与项目边界总览
summary: 说明六个注册 Project、辅助脚本和主要基础设施在英语抢词工作空间中的职责边界。
---

# 英语单词系统与项目边界总览

## 项目分工

| Project | 类型 | 核心职责 |
| --- | --- | --- |
| `rob_english_word_front` | Vue 前端 | 用户登录、抢词对战、记录、错词和掌握词 |
| `rob_english_word_cloze_web` | React 前端 | 完形任务、答题、复习、统计与音频 |
| `word_select_dashboard/web-react` | React 前端 | 运营管理、词库、任务、用户数据和配置 |
| `rob_english_word_back` | Java 后端 | 用户鉴权、实时游戏、学习数据和完形业务 |
| `word_select_dashboard/server` | Go 后端 | 管理 API、执行编排、配置和跨库查询 |
| `word_select_dashboard/word-agent` | Python 后端 | LLM 句子生成、评分、错词任务和 TTS |

## 基础设施边界

- PostgreSQL 保存用户、学习、游戏、完形、词库、句子、配置和执行记录。
- Redis 保存实时匹配、房间或临时状态，不能作为最终结算的唯一事实源。
- MinIO 或兼容对象存储保存生成音频；数据库保存业务关联和可访问引用。
- 外部 LLM、MiMo TTS 和公共词典属于外部能力，调用失败应与核心业务失败区分。
- `scripts/` 是一次性或批处理工具，不承担常驻 API，也不注册为独立 Project。

## 维护边界

- 单项目实现写入对应 Project 文档。
- 前后端或跨服务流程写入 `docs/chains/`。
- 历史方案位于 `docs/superpowers/`，只代表当时设计，不自动等同当前实现。
