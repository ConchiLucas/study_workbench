# Study Content Admin

学习内容后台。当前能力：

1. **识字**：从同库 `study_workbench` 同步识字表，自动判定是否需要义图（可人工改）
2. **配置管理**：只读消费 Shared Config Center

## 前提

- Shared Config Center 已启动（配置菜单）
- `study_workbench` Postgres 已启动（识字同步），本机常见端口 `15432`
- Docker 网络：`docker network create vibedeploy-shared`（若尚无）

## Docker 启动

```bash
make up
```

打开 http://localhost:19091 （默认进入识字）

容器通过 `host.docker.internal:15432` 连接与 workbench **同一数据库**。

## 本地开发

```bash
cd backend
SHARED_CONFIG_CENTER_BASE_URL=http://127.0.0.1:18783 \
  APP_DB_HOST=127.0.0.1 APP_DB_PORT=15432 \
  go run ./cmd/server

cd frontend
npm install && npm run dev   # http://localhost:19092
```

## 识字 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/literacy/sync` | 从题库同步 |
| GET | `/api/v1/literacy/chars` | `view=groups\|table`，可选 `needsSenseImage` |
| PATCH | `/api/v1/literacy/chars/:kpId` | `{ "needsSenseImageOverride": true\|false\|null }` |

## 设计文档

- `docs/superpowers/specs/2026-08-29-study-content-admin-config-design.md`
- `docs/superpowers/specs/2026-08-29-literacy-assets-list-design.md`
