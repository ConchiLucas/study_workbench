---
title: 英语单词后端服务文档索引
summary: 导航 Java 核心业务后端、Go 管理服务和 Python 智能代理的职责、接口、数据与协作关系。
---

# 英语单词后端服务文档索引

每个后端 Project 维护独立入口，记录运行身份、入站接口、出站依赖、逻辑数据源和关键源码锚点。

## 维护约定

- Java 服务同时区分 REST 与 Netty WebSocket；兼容接口不得写成主链路。
- Go 服务区分公开配置读取、后台鉴权 API、数据库访问和对 Python Agent 的调用。
- Python Agent 区分 HTTP 服务、CLI Runner、模型/TTS 调用、对象存储和数据库任务。
- 数据库只记录逻辑库、用途、代表表和配置位置，不记录连接凭据。

## 下级文档

| 功能说明 | 相对路径 |
| --- | --- |
| 英语单词游戏 Java 后端 | `./rob_english_word_back/AGENTS.md` |
| 单词选择 Go 管理服务 | `./word_select_dashboard_server/AGENTS.md` |
| 单词智能代理 Python 服务 | `./word_agent/AGENTS.md` |
