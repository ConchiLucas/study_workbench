---
title: 英语单词项目与逻辑数据源目录
summary: 记录业务库、缓存和对象存储的用途、所属 Project 与代码定位，不保存连接凭据。
---

# 英语单词项目与逻辑数据源目录

## PostgreSQL

| 逻辑库 | 主要使用方 | 用途 |
| --- | --- | --- |
| `rob_english_word` | Java、Go、Python Agent | 用户、单词、游戏、训练、错词、掌握词、完形任务和复习进度 |
| `select_english_word` | Go、Python Agent | 运营账号、词库清洗、句子、AI/TTS 配置、执行记录与任务状态 |

Java 的表结构和迁移线索主要位于 `rob_english_word_back/db/`；Go 的模型位于 `word_select_dashboard/server/model/`；Python 数据访问位于 `word-agent/src/word_agent/`。

## Redis

- Java 后端用于实时匹配、房间或临时游戏状态。
- 具体 Key、序列化格式、DB 编号和过期策略以源码及所选运行环境为准。

## 对象存储

- Go 与 Python Agent 使用 MinIO Client 管理 TTS 音频等对象。
- 文档只记录业务用途和代码入口，不保存 endpoint、access key 或 secret key。

## Context Router 别名

数据库查询必须使用当前 `prepare_task_context` 返回的稳定别名。别名代表授权范围，不等同于物理连接信息。
