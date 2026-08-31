---
title: 英语单词开发工作空间文档入口
summary: 定义英语抢词、完形练习、运营后台和智能代理项目的任务定位、文档导航与跨项目协作规则。
---

# 英语单词开发工作空间文档入口

本工作空间包含用户侧英语抢词应用、完形填空练习、运营管理后台、Java/Go 后端和 Python 智能代理，不是单一应用。

## 如何使用本工作空间

1. 先判断任务属于用户主站、完形练习、运营后台、Java 核心后端、Go 管理服务还是 Python Agent。
2. 单项目任务先读取对应 `docs/frontend|backend/<project>/AGENTS.md`，再回到源码核对。
3. 涉及登录、实时对战、完形复习、错词进度或 AI/TTS 时，先读 `docs/chains/README.md` 下的端到端链路。
4. 涉及启动、代理、数据库、Redis、对象存储或外部模型时，先读 `docs/shared/README.md`。
5. 文档用于定位和解释；最终实现、配置和运行行为以当前源码与受控环境为准。

## Context Router MCP

- 新任务涉及业务规则、启动、数据库、环境配置或跨项目链路时，先调用 `prepare_task_context`，保存返回的 `task_id`。
- 目标不明确时使用 `search_context_documents`；目标明确时用 `read_context_document` 只读必要文档或章节。
- 数据库对象不明确时先搜索 Schema，再执行单条、有界、只读查询。
- prepare 返回的数据库别名和环境配置只用于当前任务，不在回答、日志、源码或文档中回显凭据。
- 用户说“启动”“启动项目”或“启动服务”时，统一调用 `start_workspace(task_id)`；它始终启动本工作空间登记的全部项目，不拆分询问或只启动当前子项目。
- 完成代码或受版本控制配置修改后，收集本轮所有真实 Workspace 相对路径，一次调用 `apply_workspace_changes(task_id, changed_files)`；不要传未修改路径，也不要按 Project 拆成多次调用。
- 单 Project 普通源码改动应只生成该 Project 的一个 `fast` 步骤；依赖、构建、Compose、Workspace 级文件或跨 Project 改动可升级为 `full` 或 Workspace 启动步骤，以返回的 `decision_reason` 为准。
- `apply_workspace_changes` 与 `start_workspace` 返回 `operation_id` 后，调用 `get_workspace_operation(operation_id)` 轮询到 `succeeded`、`failed`、`cancelled` 或 `interrupted`；服务端会从操作记录校验所属任务，再向用户报告客观结果。
- 终态为 `failed`、`cancelled` 或 `interrupted` 时，保留已启动容器和日志；除非用户明确要求，不自动修复目标配置，也不清理容器、镜像、Volume 或业务数据。
- `.env.local` 是当前电脑唯一私有运行配置入口，不读取、不回显、不写入 Context Router；只有用户明确要求修改本机配置时才编辑它，变更后仍通过 Workspace 启动验证。
- MCP 不可用或映射尚未刷新时，可按本文档树直接读取本地 Markdown 与源码，不阻塞任务。
- 仓库内运行配置以 `deploy/context-router/` 为唯一事实源；在 Context Router 的 Workspace 页面使用“同步 deploy 配置”扫描并原子替换数据库运行配置。
- Context Router 不可用时，先阅读 `deploy/context-router/README.md`；启动全部项目可直接执行 `deploy/context-router/workspace/start/deploy.sh`，单 Project 的入口位于其项目根 `deploy/context-router/fast|full/deploy.sh`。

## 文档维护约定

- 根文档只维护工作空间级规则和导航，不重复单项目实现。
- 单项目事实写入对应 Project 的 `AGENTS.md`；跨项目完整流程写入 `docs/chains/`；公共运行与依赖事实写入 `docs/shared/`。
- 父子关系只由 `## 下级文档` 下的“功能说明 / 相对路径”两列表格定义。
- 路径必须以 `./` 开头、指向 Markdown，并保持在父文档所在目录内。
- 无法静态确认的行为标记为“待运行核对”，不把推断写成既成事实。
- 不记录密码、Token、私钥、完整连接串或其他敏感值。

## 下级文档

| 功能说明 | 相对路径 |
| --- | --- |
| 后端服务文档索引 | `./docs/backend/README.md` |
| 前端项目文档索引 | `./docs/frontend/README.md` |
| 公共技术文档索引 | `./docs/shared/README.md` |
| 跨项目业务链路索引 | `./docs/chains/README.md` |

## 工作空间操作约束

- 修改代码前先确认目标子项目、技术栈、构建方式和影响链路。
- 修改接口、WebSocket 消息、数据模型、代理或环境配置后，同步复核关联项目和链路文档。
- `docs/superpowers/` 保存历史设计与实施计划，不自动纳入当前事实树；需要时按任务检索读取。
- `scripts/` 是辅助工具目录，不作为第七个常驻 Project；相关说明归入公共运行文档。
