---
title: 英语单词游戏 Java 后端
summary: 提供用户鉴权、单词学习、实时抢词对战、训练记录、错词掌握和完形练习等核心业务能力。
---

# 英语单词游戏 Java 后端

## 职责与边界

- 负责普通用户注册登录、JWT 鉴权、用户资料和英语学习数据。
- 负责抢词匹配、房间、出题、答题、机器人、结算和训练记录。
- 负责错词、已掌握单词、完形任务、复习进度、答题历史与统计。
- 运营后台、AI Provider 配置和句子生成执行编排由 Go 服务与 Python Agent 负责。

## 项目定位

| 项目 | 值 |
| --- | --- |
| 相对路径 | `rob_english_word_back` |
| 技术栈 | Java 21、Spring Boot 3.2、Spring Security、MyBatis-Plus、PostgreSQL、Redis、Netty |
| 构建入口 | `pom.xml`，Maven artifact `rob-english-word` |
| 启动类 | `src/main/java/com/robword/RobEnglishWordApplication.java` |
| 主要配置 | `src/main/resources/application.yml`、根目录及项目目录 Compose |

## 入站接口

- 鉴权：`/api/auth/register`、`/api/auth/login`。
- 用户：`/api/user/info`、`/api/user/{userId}`、`/api/user/profile`。
- 学习记录：`/api/game/records`、`/api/game/records/{recordId}`、`/api/game/answer-detail`。
- 错词与掌握：`/api/wrong-words/**`、`/api/mastered-words`。
- 完形练习：`/api/cloze-practice/tasks/**`、`preferences/**`、`answers`、`history`、`stats`。
- 外部完形生成入口：`/api/external/sentence-cloze/generate`。
- 实时通信：Netty WebSocket `/ws?token=...`，消息由 `GameChannelHandler` 分发。

`/api/match/start` 和 `/api/match/cancel` 只保留兼容提示；当前匹配主路径是 WebSocket 的 `match_start`、`match_cancel` 消息。

## 核心模块与数据

- `controller/`：REST 入口；`config/`：Security、异常和 Web 配置。
- `netty/`：WebSocket Server、Pipeline、Channel 管理和游戏消息处理。
- `service/`：鉴权、匹配、游戏状态机、答题、结算、错词与完形业务。
- `mapper/`、`entity/`：MyBatis-Plus 数据访问与持久化模型。
- 逻辑业务库为 `rob_english_word`；代表表定义位于 `db/`，覆盖用户、单词、训练、错词、掌握词、完形任务和复习进度。
- Redis 承载匹配、房间或临时状态；持久结果写入 PostgreSQL。具体 Key 和过期策略以源码与运行配置为准。

## 关键代码锚点

- 鉴权：`src/main/java/com/robword/controller/AuthController.java`
- 游戏：`src/main/java/com/robword/service/GameService.java`
- 答题与结算：`src/main/java/com/robword/service/AnswerService.java`、`GameSettlementService.java`
- WebSocket：`src/main/java/com/robword/netty/NettyWebSocketServer.java`、`GameChannelHandler.java`
- 错词：`src/main/java/com/robword/controller/WrongWordController.java`
- 完形：`src/main/java/com/robword/controller/ClozePracticeController.java`
- 数据脚本：`db/`

## 关联链路

- [登录注册与鉴权](../../chains/login-authentication.md)
- [实时抢词对战](../../chains/realtime-match-game.md)
- [完形练习与复习](../../chains/cloze-practice-review.md)
- [错词与掌握进度](../../chains/wrong-word-mastery.md)

## 运行时待核对

- WebSocket 在实际部署中的 Nginx 升级头、端口映射和断线重连表现。
- Redis 中房间状态与 PostgreSQL 结算事务在异常中断时的一致性。
- 外部完形生成入口的调用方、超时与失败重试策略。

## 复核信息

- 复核日期：2026-08-01；源码基线：`dev` 分支。
- 证据：`pom.xml`、启动类、Controller、Netty、Service、Mapper、`db/` 和部署文件。
