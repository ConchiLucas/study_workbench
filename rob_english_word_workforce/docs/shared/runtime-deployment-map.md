---
title: 英语单词本地运行与部署目录说明
summary: 区分根 Compose、各项目 Compose、集中 deploy 目录和辅助重启脚本的适用范围。
---

# 英语单词本地运行与部署目录说明

## 运行入口

- Docker 全量启动的唯一配置入口是仓库根 `.env.local`，唯一正式命令是根 `deploy-compose-full.sh`。
- 新电脑首次运行时复制根 `.env.example` 为 `.env.local`，填写该电脑实际可访问的 PostgreSQL、Redis、MinIO 和可选 CLI 路径。不要在六个子项目中分别维护连接配置。
- 根 `docker-compose.yml` 主要组合 Vue 用户前端、Java 后端、PostgreSQL 和 Redis。
- 各子项目自己的 `docker-compose.yml` 用于该项目独立运行或联调。
- `deploy/frontend/` 与 `deploy/backend/` 保存六个 Project 的集中部署配置，区分快速或完整更新模式。
- 根目录 `restart_all_services.sh`、`restart_service_common.sh` 及各项目重启脚本是运维辅助入口。
- `scripts/` 保存一次性或批处理工具，运行前必须先阅读脚本参数和数据影响。

## Context Router 运行编排

- Context Router 作为控制面保存 Workspace `start` 包装器、Workspace 级路径、六个 Project 的执行顺序，以及各 Project 的 `fast/full` 部署快照；它不保存本机连接值。
- Context Router Host Runtime Runner 由用户在 Context Router 仓库手动启动，不使用 Docker/launchd 开机自启动。它只领取服务端已物化的不可变快照，验证 manifest 后执行固定 `deploy.sh`。
- 本 Workspace 的 `start/deploy.sh` 只允许调用 `"$WORKSPACE_HOST_ROOT/deploy-compose-full.sh"`。根全量脚本负责读取 `.env.local`、校验当前电脑差异并启动全部六个项目。
- Codex 在本 Workspace 任意子目录收到启动请求时调用 `start_workspace`；代码修改完成后调用一次 `apply_workspace_changes`。两种调用都必须用 `get_workspace_operation` 等待终态。
- Project 源码变更按最长源码前缀路由；依赖、锁文件、Dockerfile 或 Compose 变更使用 `full`，其他业务源码使用 `fast`。根部署入口、`.env.example` 和登记的公共运行脚本变化使用 Workspace `start`。
- `.env.local` 继续只存在目标仓库且被 Git 忽略；其 PostgreSQL、Redis、MinIO 与本机 CLI 路径只由目标根脚本加载，不进入 MCP 参数、Context Router 数据库、快照或日志摘要。

## Docker 全量启动

```bash
cp .env.example .env.local
$EDITOR .env.local
./deploy-compose-full.sh --check
./deploy-compose-full.sh
```

`.env.local` 是每台电脑唯一需要维护的私有运行配置，并被 Git 忽略。脚本会从自身位置解析仓库根目录，因此可以从任意当前目录调用。

配置中的 Host 表示宿主机可访问地址。值为 `127.0.0.1`、`localhost` 或 `::1` 时，脚本自动为 Docker 容器派生 `host.docker.internal`；远程 IP 和域名保持不变。用户不需要分别填写宿主机 DSN 和容器 DSN。

`word_select_dashboard/word-agent/.env` 与 `word_select_dashboard/server/config.yaml` 继续服务于历史原生启动兼容，不属于 Docker 全量启动的配置入口。全量脚本不会从它们读取机器差异。

`--check` 在构建前验证根配置、六份 Compose、外部依赖地址和可选 CLI 路径；成功后不会创建镜像或容器。正式启动还会启动宿主机 CLI Runner，并验证六个目标容器、三个后端/Agent 和三个前端。

## 推荐核对顺序

1. 明确要启动的是用户主站组合、单个 Project，还是全部六个 Project。
2. 核对镜像构建上下文、挂载文件、环境变量来源和依赖服务。
3. 启动后分别验证静态页面、REST 健康检查、WebSocket、数据库、Redis 和 Agent 外部依赖。
4. 不把“容器处于 running”当作业务可用；至少执行对应健康检查或代表性只读请求。

## 安全约束

- 文档不记录真实密码、Token、模型 Key 或对象存储密钥。
- 部署文件包含敏感值时，应通过环境变量或本机受控配置注入，并限制版本库暴露。
- 修改端口、代理或服务名后同步更新请求寻址与相关链路文档。
