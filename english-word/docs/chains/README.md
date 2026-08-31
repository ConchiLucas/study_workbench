---
title: 英语单词跨项目业务链路索引
summary: 汇总登录鉴权、实时对战、完形复习、错词掌握和 AI 句子生成等端到端流程。
---

# 英语单词跨项目业务链路索引

本目录维护跨 Project、跨前后端或跨存储的完整业务流程。单项目内部实现仍写入对应项目文档。

## 维护约定

- 按步骤记录页面、请求或消息、后端入口、服务处理、数据落点和证据路径。
- 明确区分源码静态确认、配置推断和必须运行验证的行为。
- 链路变化后同步复核涉及的 Project 文档，避免同一事实出现冲突版本。

## 下级文档

| 功能说明 | 相对路径 |
| --- | --- |
| 登录注册与 JWT 鉴权链路 | `./login-authentication.md` |
| WebSocket 匹配、对战与结算链路 | `./realtime-match-game.md` |
| 完形任务、答题与复习链路 | `./cloze-practice-review.md` |
| 错词、掌握词与学习进度链路 | `./wrong-word-mastery.md` |
| AI 句子生成、评分与 TTS 链路 | `./ai-sentence-tts-scoring.md` |
